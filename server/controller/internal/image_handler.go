package internal

import (
	"archive/tar"
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/openpgp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/common"
	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
)

const maxImageSize int64 = 1 << 30
const manifestFile = "manifest.json"

func imageDir() string {
	return common.Config.Controller.ImageDir
}

func imageSigner() string {
	return common.Config.Controller.ImageSigner
}

type ImageServer struct {
	pb.UnimplementedImageServer
}

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	images, err := model.ListImages()
	if err != nil {
		log.Warnf("Failed to get images list: %v", err)
		return nil, rpc.ErrInternal
	}

	reply := pb.ListReply{}
	for _, image := range images {
		if image.ApprovalStatus != model.ApprovalPass {
			continue
		}
		reply.Images = append(reply.Images, &pb.ImageInfo{
			Name: image.Name + ":" + image.Version,
			Repo: image.Name,
			Tag:  image.Version,
			Size: image.FileSize,
		})
	}

	return &reply, nil
}

func (s *ImageServer) ListDB(ctx context.Context, in *pb.ListDBRequest) (*pb.ListDBReply, error) {
	reply := pb.ListDBReply{}
	images, err := model.ListImages()
	if err != nil {
		log.Warnf("Failed to get images list: %v", err)
		return nil, rpc.ErrInternal
	}

	for _, image := range images {
		reply.Images = append(reply.Images, &pb.ImageDBInfo{
			Id:             image.ID,
			Name:           image.Name,
			Version:        image.Version,
			Description:    image.Description,
			VerifyStatus:   image.VerifyStatus,
			ApprovalStatus: image.ApprovalStatus,
			Size:           image.FileSize,
			UpdateAt:       image.UpdatedAt,
		})
	}

	return &reply, nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		log.Errorf("request is canceled")
		return rpc.ErrCanceled
	case context.DeadlineExceeded:
		log.Errorf("deadline is exceeded")
		return rpc.ErrDeadlineExceeded
	default:
		return nil
	}
}

func getHash(filename, sendHashStr string) error {
	file, err := os.Open(filename)
	if err != nil {
		log.Errorf("open file %v err: %v", filename, err)
		return err
	}
	defer file.Close()

	hash256 := sha256.New()
	_, err = io.Copy(hash256, file)
	if err != nil {
		log.Errorf("io copy error: %v", err)
		return err
	}

	hashStr := hex.EncodeToString(hash256.Sum(nil))
	if hashStr != sendHashStr {
		log.Errorf("file hash not match: [%v] != [%v]", hashStr, sendHashStr)
		return rpc.ErrInternal
	}

	return nil
}

func getImageID(imageFile string) string {
	file, err := os.Open(imageFile)
	if err != nil {
		log.Printf("open image file %v err: %v", imageFile, err)
		return ""
	}
	defer file.Close()

	reader := tar.NewReader(file)
	for header, err := reader.Next(); err != io.EOF; header, err = reader.Next() {
		if err != nil {
			log.Printf("reader next err: %v", err)
			return ""
		}

		fileName := header.FileInfo().Name()
		if fileName == manifestFile {
			continue
		}

		index := strings.Index(fileName, ".json")
		if index == -1 {
			continue
		}

		return fileName[0:index]
	}

	return ""
}

func signVerify(sigFile, tarFile string) int32 {
	keyRingReader, err := os.Open(imageSigner())
	if err != nil {
		log.Warnf("open file=%v err=%v", imageSigner(), err)
		return model.VerifyAbnormal
	}
	keyring, err := openpgp.ReadKeyRing(keyRingReader)
	if err != nil {
		log.Warnf("read key ring err=%v", err.Error())
		return model.VerifyAbnormal
	}

	sig, err := os.Open(sigFile)
	if err != nil {
		log.Warnf("open file=%v err=%v", sigFile, err)
		return model.VerifyAbnormal
	}

	tar, err := os.Open(tarFile)
	if err != nil {
		log.Warnf("open file=%v err=%v", tarFile, err)
		return model.VerifyAbnormal
	}

	_, err = openpgp.CheckDetachedSignature(keyring, tar, sig)
	if err != nil {
		log.Infof("check signature tar=%v sig=%v err=%v", tarFile, sigFile, err)
		return model.VerifyFail
	}

	return model.VerifyPass
}

func (s *ImageServer) Upload(stream pb.Image_UploadServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Errorf("cannot receive image info %v", err)
		return rpc.ErrUnknown
	}

	if req.Info == nil || req.Sign == nil {
		return rpc.ErrInvalidArgument
	} else if !isValidImageName(req.Info.Name) {
		return status.Errorf(codes.InvalidArgument, "镜像名参数错误")
	} else if !isValidImageVersion(req.Info.Version) {
		return status.Errorf(codes.InvalidArgument, "镜像版本参数错误")
	} else if !isValidImageDesc(req.Info.Description) {
		return status.Errorf(codes.InvalidArgument, "镜像描述参数错误")
	}

	fileName := fmt.Sprintf("%s/%s_%s%s", imageDir(), req.Info.Name, req.Info.Version, req.Info.Type)
	signFileName := fmt.Sprintf("%s/%s_%s.sign", imageDir(), req.Info.Name, req.Info.Version)
	{
		signfile, err := os.Create(signFileName)
		if err != nil {
			log.Errorf("cannot create file %v: %v", signFileName, err)
			return rpc.ErrInternal
		}
		defer signfile.Close()

		req, err := stream.Recv()
		if err != nil {
			log.Errorf("recv sign err: %v", err)
			return err
		}

		size := len(req.ChunkData)
		if req.Sign.Size != int64(size) {
			log.Errorf("recv sign size err, send[%v] recv:[%v]", req.Sign.Size, size)
			return rpc.ErrInternal
		}

		_, err = signfile.Write(req.ChunkData)
		if err != nil {
			log.Errorf("cannot write chunk data to file: %v", err)
			return rpc.ErrInternal
		}
	}

	file, err := os.Create(fileName)
	if err != nil {
		log.Errorf("cannot create file %v: %v", fileName, err)
		return rpc.ErrInternal
	}
	defer file.Close()

	var imageSize int64
	for {
		if err := contextError(stream.Context()); err != nil {
			return err
		}

		req, err := stream.Recv()
		if err == io.EOF {
			log.Infof("no more data")
			break
		}
		if err != nil {
			log.Errorf("cannot receive chunk data: %v", err)
			return rpc.ErrUnknown
		}

		size := len(req.ChunkData)

		imageSize += int64(size)
		if imageSize > maxImageSize || imageSize > req.Info.Size {
			log.Errorf("image is too large: [%v] > [%v] || [%v]", imageSize, maxImageSize, req.Info.Size)
			return rpc.ErrInvalidArgument
		}
		// write slowly
		time.Sleep(time.Millisecond)
		_, err = file.Write(req.ChunkData)
		if err != nil {
			log.Errorf("cannot write chunk data to file: %v", err)
			return rpc.ErrInternal
		}
	}
	/*
		if err = getHash(fileName, req.Info.Checksum); err != nil {
			tmpErr := os.Remove(fileName)
			log.Warnf("hash err and remove image file %v: %v", fileName, tmpErr)
			return err
		}
	*/

	verifyStatus := signVerify(signFileName, fileName)

	imaegId := getImageID(fileName)
	if imaegId == "" {
		log.Warnf("the image id of image file %v is wrong ", fileName)
		return rpc.ErrInvalidArgument
	}

	log.Printf("imaegId: [%v]", imaegId)

	imageInfo := model.ImageInfo{
		Name:         req.Info.Name,
		Version:      req.Info.Version,
		Description:  req.Info.Description,
		FileSize:     req.Info.Size,
		FileType:     req.Info.Type,
		CheckSum:     req.Info.Checksum,
		ImageId:      imaegId,
		FilePath:     fileName,
		SignPath:     signFileName,
		VerifyStatus: verifyStatus,
	}

	imageId, err := model.CreateImages(imageInfo)
	if err != nil {
		return rpc.ErrInternal
	}

	res := &pb.UploadReply{
		ImageId: imageId,
	}

	err = stream.SendAndClose(res)
	if err != nil {
		log.Errorf("cannot send response: %v", err)
		return rpc.ErrUnknown
	}

	log.Debugf("Upload image %v ok", imageId)

	return nil
}

func (s *ImageServer) Update(stream pb.Image_UpdateServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Errorf("cannot receive image info %v", err)
		return rpc.ErrUnknown
	}

	if req.Info == nil || (req.Info.Size != 0 && req.Sign == nil) || (req.Info.Size == 0 && req.Sign != nil) {
		return rpc.ErrInvalidArgument
	} else if !isValidImageName(req.Info.Name) {
		return status.Errorf(codes.InvalidArgument, "镜像名参数错误")
	} else if !isValidImageVersion(req.Info.Version) {
		return status.Errorf(codes.InvalidArgument, "镜像版本参数错误")
	} else if !isValidImageDesc(req.Info.Description) {
		return status.Errorf(codes.InvalidArgument, "镜像描述参数错误")
	}

	img, err := model.QueryImageByID(req.ImageId)
	if err != nil {
		return err
	}

	var fileName, signFileName string
	if req.Sign != nil && req.Sign.Size != 0 {
		signFileName = fmt.Sprintf("%s/%s_%s.sign", imageDir(), req.Info.Name, req.Info.Version)
		signfile, err := os.Create(signFileName)
		if err != nil {
			log.Errorf("cannot create file %v: %v", signFileName, err)
			return rpc.ErrInternal
		}
		defer signfile.Close()

		req, err := stream.Recv()
		if err != nil {
			log.Errorf("recv sign err: %v", err)
			return err
		}

		chunk := req.ChunkData
		size := len(chunk)
		if req.Sign.Size != int64(size) {
			log.Errorf("recv sign size err, send[%v] recv:[%v]", req.Sign.Size, size)
			return rpc.ErrInternal
		}

		_, err = signfile.Write(chunk)
		if err != nil {
			log.Errorf("cannot write chunk data to file: %v", err)
			return rpc.ErrInternal
		}
	}
	if req.Info.Size != 0 {
		fileName = fmt.Sprintf("%s/%s_%s%s", imageDir(), req.Info.Name, req.Info.Version, req.Info.Type)
		file, err := os.Create(fileName)
		if err != nil {
			log.Errorf("cannot create file: %v", fileName)
			return rpc.ErrInternal
		}
		defer file.Close()

		var imageSize int64
		for {
			err := contextError(stream.Context())
			if err != nil {
				return err
			}

			req, err := stream.Recv()
			if err == io.EOF {
				log.Infof("no more data")
				break
			}
			if err != nil {
				log.Errorf("cannot receive chunk data: %v", err)
				return rpc.ErrUnknown
			}

			size := len(req.ChunkData)
			imageSize += int64(size)
			if imageSize > maxImageSize || imageSize > req.Info.Size {
				log.Errorf("image is too large: [%v] > [%v] || [%v]", imageSize, maxImageSize, req.Info.Size)
				return rpc.ErrInvalidArgument
			}
			// write slowly
			time.Sleep(time.Millisecond)
			_, err = file.Write(req.ChunkData)
			if err != nil {
				log.Errorf("cannot write chunk data to file: %v", err)
				return rpc.ErrInternal
			}
		}
	}
	/*
		if err = getHash(fileName, req.Info.Checksum); err != nil {
			tmpErr := os.Remove(fileName)
			log.Warnf("hash err and remove image file %v: %v", fileName, tmpErr)
			return err
		}
	*/

	if fileName != "" && signFileName != "" {
		verifyStatus := signVerify(signFileName, fileName)
		imaegId := getImageID(fileName)
		if imaegId == "" {
			log.Warnf("the image id of image file %v is wrong ", fileName)
			return rpc.ErrInvalidArgument
		}

		log.Printf("imaegId: [%v]", imaegId)
		if img.FilePath != fileName {
			err = os.Remove(img.FilePath)
			log.Debugf("db update and remove old image file %v: %v", img.FilePath, err)
		}

		if img.SignPath != signFileName {
			err = os.Remove(img.SignPath)
			log.Debugf("db update and remove old image sign %v: %v", img.SignPath, err)
		}

		img.FileSize = req.Info.Size
		img.CheckSum = req.Info.Checksum
		img.ImageId = imaegId
		img.FilePath = fileName
		img.SignPath = signFileName
		img.VerifyStatus = verifyStatus
	}

	img.Description = req.Info.Description

	err = model.UpdateImage(img)
	if err != nil {
		return rpc.ErrInternal
	}

	reply := pb.UpdateReply{}
	err = stream.SendAndClose(&reply)
	if err != nil {
		log.Errorf("cannot send response: %v", err)
		return rpc.ErrUnknown
	}

	log.Debugf("Update image %v ok", req.ImageId)
	return nil
}

func (s *ImageServer) Download(in *pb.DownloadRequest, stream pb.Image_DownloadServer) error {
	imageInfo, err := model.QueryImageByID(in.ImageId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return rpc.ErrNotFound
		}
		return rpc.ErrInternal
	}

	if err = getHash(imageInfo.FilePath, imageInfo.CheckSum); err != nil {
		// tmpErr := model.RemoveImage([]int64{in.ImageId})
		// log.Warnf("image file %v hash %v err and remove, remove result:%v", imageInfo.FilePath, imageInfo.CheckSum, tmpErr)
		return status.Errorf(codes.Internal, "镜像文件被破坏")
	}

	file, err := os.Open(imageInfo.FilePath)
	if err != nil {
		return rpc.ErrInternal
	}
	defer file.Close()

	reply := &pb.DownloadReply{
		Info: &pb.UploadInfo{
			Name:     imageInfo.Name,
			Version:  imageInfo.Version,
			Type:     imageInfo.FileType,
			Checksum: imageInfo.CheckSum,
			Size:     imageInfo.FileSize,
		},
	}
	err = stream.Send(reply)
	if err != nil {
		log.Errorf("send image info fail: %v", err)
		return rpc.ErrInternal
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024*1024)
	for {
		n, err := reader.Read(buffer)

		if err == io.EOF {
			log.Infof("no more data")
			break
		}
		if err != nil {
			log.Errorf("read image file fail: %v", err)
			return rpc.ErrInternal
		}

		reply := &pb.DownloadReply{
			ChunkData: buffer[:n],
		}

		if err = stream.Send(reply); err != nil {
			log.Errorf("Send image file fail: %v", err)
			return rpc.ErrInternal
		}
	}

	return nil
}

func (s *ImageServer) noticeAgentSync(toRemove, toPull []string) {
	nodes, err := model.ListNodes()
	if err != nil {
		log.Warnf("noticeAgentSync list nodes err=%v", err)
		return
	}

	// 遍历各节点通过ImagePull拉取镜像
	for _, n := range nodes {
		conn, err := getAgentConn(n.Address)
		if err != nil {
			log.Warnf("Failed to connect to agent service, node=%+v", n)
			continue
		}

		req := pb.AgentSyncRequest{
			ToRemove: toRemove,
			ToPull:   toPull,
		}

		cli := pb.NewImageClient(conn)
		for i := 0; i < 3; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			defer cancel()
			if _, err = cli.AgentSync(ctx, &req); err != nil {
				log.Infof("sync image %+v on node address=%v err=%v", &req, n.Address, err)
				continue
			}

			log.Debugf("sync image %+v on node address=%v", &req, n.Address)
			break
		}
	}
}

func (s *ImageServer) Approve(ctx context.Context, in *pb.ApproveRequest) (*pb.ApproveReply, error) {
	i, err := model.QueryImageByID(in.ImageId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "镜像不存在")
		}
		return nil, rpc.ErrDatabaseFail
	}

	if in.Approve {
		if i.VerifyStatus != model.VerifyPass {
			return nil, status.Errorf(codes.FailedPrecondition, "镜像校验未通过，无法审批通过")
		}
		i.ApprovalStatus = model.ApprovalPass
	} else {
		if in.RejectReason == "" && i.VerifyStatus == model.VerifyPass {
			return nil, status.Errorf(codes.InvalidArgument, "参数错误：拒绝原因不能为空")
		}
		i.ApprovalStatus = model.ApprovalReject
	}
	i.RejectReason = in.RejectReason

	if err := model.UpdateImage(i); err != nil {
		log.Infof("Approve image=%+v err=%v", i, err)
		return nil, rpc.ErrDatabaseFail
	}

	// 审批通过后 推送registry 通知agent同步
	if in.Approve {
		go func() {
			model.PushImage(i)
			s.noticeAgentSync(nil, []string{i.Name + ":" + i.Version})
		}()
	}

	return &pb.ApproveReply{}, nil
}

func (s *ImageServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	images, err := model.QueryImageByIDs(in.ImageIds)
	if err != nil {
		log.Infof("DB query image info err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	if err := model.RemoveImages(in.ImageIds); err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	go func() {
		// remove in registry, node agent remove local image
		var toRemove []string
		for _, i := range images {
			v := i.Name + ":" + i.Version
			if err := model.RemoveRegistryImage(v); err != nil {
				log.Infof("registry remove image name=%v version=%v, err=%v", i.Name, i.Version, err)
			}
		}
		s.noticeAgentSync(toRemove, nil)
	}()

	return &pb.RemoveReply{}, nil
}

// CronSyncImage 定时执行函数cleanRegistryImages和syncNodeImage
func CronSyncImage() {
	for {
		if isMaster() {
			SyncNodeImages()
		}
		time.Sleep(time.Minute)
	}
}

// SyncNodeImages 读取数据库中镜像列表，删除节点上多余的（注意不能删除备份产生的镜像），
// 拉取缺失的
func SyncNodeImages() {
	validImages, err := allValidImages()
	if err != nil {
		log.Infof("get valid images err=%v", err)
		return
	}

	nodes, err := model.ListNodes()
	if err != nil {
		log.Warnf("Failed to list node images: %v", err)
		return
	}

	for _, n := range nodes {
		go syncNodeImages(validImages, n.Address)
	}
}

func syncNodeImages(validImages map[string]int, addr string) {
	conn, err := getAgentConn(addr)
	if err != nil {
		log.Warnf("Failed to connect to agent service, addr=%v", addr)
		return
	}

	cli := pb.NewImageClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	reply, err := cli.List(ctx, &pb.ListRequest{})
	if err != nil {
		log.Warnf("image.List for node=%v err=%v", addr, err)
		return
	}

	nodeImages := make(map[string]bool, len(reply.Images))
	for _, i := range reply.Images {
		nodeImages[i.Name] = true
	}

	var req pb.AgentSyncRequest

	// TODO 优化备份创建数据维护，防止备份镜像不会被错误地删除后取消下面的注释
	// 节点存在的镜像，上传和备份中都没有，需要删除
	// for image := range nodeImages {
	// 	if _, ok := validImages[image]; !ok {
	// 		req.ToRemove = append(req.ToRemove, image)
	// 	}
	// }

	// 上传且审批通过的镜像，节点中不存在，需要同步
	for image, t := range validImages {
		_, ok := nodeImages[image]
		if !ok && t == ImageSourceUpload {
			req.ToPull = append(req.ToPull, image)
		}
	}

	if len(req.ToRemove) == 0 && len(req.ToPull) == 0 {
		return // no changes
	}

	ctx, cancel = context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	_, err = cli.AgentSync(ctx, &req)
	if err != nil {
		log.Warnf("image.AgentSync for node=%v err=%v", addr, err)
		return
	}
}
