package internal

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"scmc/model/db"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
	"scmc/server/args"
)

type ImageUploadStream struct {
	ss pb.Image_UploadServer
}

func (s ImageUploadStream) Read() ([]byte, error) {
	r, err := s.ss.Recv()
	if err != nil {
		return nil, err
	}

	return r.ChunkData, nil
}

func (s ImageUploadStream) Context() context.Context {
	return s.ss.Context()
}

type ImageUpdateStream struct {
	ss pb.Image_UpdateServer
}

func (s ImageUpdateStream) Read() ([]byte, error) {
	r, err := s.ss.Recv()
	if err != nil {
		return nil, err
	}

	return r.ChunkData, nil
}

func (s ImageUpdateStream) Context() context.Context {
	return s.ss.Context()
}

type ImageServer struct {
	pb.UnimplementedImageServer
}

func (s *ImageServer) noticeAgentSync(removes []string, new *db.ImageInfo) {
	nodes, err := db.ListNodes()
	if err != nil {
		log.Errorf("db.ListNodes err=%v", err)
		return
	}

	loads := []*pb.LoadInfo{}
	if new != nil {
		loads = append(loads, &pb.LoadInfo{
			ImageId: new.ImageId,
			Path:    new.FilePath,
			Tag:     new.Name + ":" + new.Version,
		})
	}

	// 遍历各节点通过ImagePull拉取镜像
	for _, n := range nodes {
		conn, err := GetAgentConn(n.Address)
		if err != nil {
			continue
		}

		req := pb.AgentSyncRequest{
			ToRemove: removes,
			ToLoad:   loads,
		}

		if _, err = pb.NewImageClient(conn).AgentSync(context.Background(), &req); err != nil {
			log.Warnf("AgentSync req=%+v node=%+v err=%v", &req, n, err)
		} else {
			log.Debugf("AgentSync req=%+v node=%+v OK", &req, n)
		}
	}
}

func (s *ImageServer) noticeNewImage(i *db.ImageInfo) {
	if i.VerifyStatus == db.VerifyPass && i.ApprovalStatus == db.ApprovalPass {
		go func() {
			s.noticeAgentSync(nil, i)
		}()
	}
}

func (s *ImageServer) uploadCheckMetadata(md *pb.UploadInfo) error {
	if md == nil {
		return rpc.ErrInvalidArgument
	} else if md.Size == 0 {
		return rpc.ErrInvalidArgument
	} else if md.Size > imageMaxSize() {
		return rpc.ErrImageSize(float64(imageMaxSize()))
	} else if !args.ImageName.Check(md.Name) {
		return status.Errorf(codes.InvalidArgument, "镜像名参数错误")
	} else if !args.ImageVersion.Check(md.Version) {
		return status.Errorf(codes.InvalidArgument, "镜像版本参数错误")
	} else if !args.ImageDesc.Check(md.Description) {
		return status.Errorf(codes.InvalidArgument, "镜像描述参数错误")
	}
	return nil
}

func (s *ImageServer) updateCheckMetadata(md *pb.UploadInfo) error {
	if md == nil {
		return rpc.ErrInvalidArgument
	} else if !args.ImageDesc.Check(md.Description) {
		return status.Errorf(codes.InvalidArgument, "镜像描述参数错误")
	}

	if md.Size > 0 && md.Size > imageMaxSize() {
		return rpc.ErrImageSize(float64(imageMaxSize()))
	}

	return nil
}

func (s *ImageServer) receiveImageFile(securitySwitch bool, img *db.ImageInfo, sig *pb.SignInfo, stream ImageFileStream) (*ImageFile, error) {
	if securitySwitch { // 开启安全开关开启时，需要文件签名
		if sig == nil || len(sig.ChunkData) == 0 {
			return nil, status.Error(codes.InvalidArgument, "缺少镜像签名")
		} else if len(sig.ChunkData) > 8192 {
			return nil, status.Error(codes.InvalidArgument, "镜像签名数据过大")
		}
	}

	f, err := NewImageFile(img.Name, img.Version, img.FileSize)
	if err != nil {
		log.Errorf("NewImageFile err=%v", err)
		return nil, rpc.ErrInternal
	}

	if err := f.ReadStream(stream); err != nil {
		f.Destroy()
		log.Errorf("Write stream data to image file err=%v", err)
		return nil, status.Errorf(codes.InvalidArgument, "读写镜像数据失败：%s", err.Error())
	}

	// verify sha256 checksum
	sum := f.SHA256()
	if sum != img.CheckSum {
		f.Destroy()
		log.Warnf("File sha256sum=%v expected=%v, not match", sum, img.CheckSum)
		return nil, status.Error(codes.InvalidArgument, "CHECKSUM匹配失败")
	}

	// verify signature if neccessary
	if securitySwitch {
		ok, err := f.VerifySignature(sig.ChunkData)
		if err != nil {
			img.VerifyStatus = db.VerifyAbnormal
		} else if !ok {
			img.VerifyStatus = db.VerifyFail
		} else {
			img.VerifyStatus = db.VerifyPass
		}
		img.ApprovalStatus = db.ApprovalWait
		img.Signature = sig.ChunkData
	} else {
		img.VerifyStatus = db.VerifyPass
		img.ApprovalStatus = db.ApprovalPass
	}

	img.FilePath = f.Path()
	img.ImageId = f.DockerID()

	f.Close()
	return f, nil
}

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	images, err := db.ListImages()
	if err != nil {
		log.Warnf("Failed to get images list: %v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListReply
	for _, i := range images {
		if i.ApprovalStatus != db.ApprovalPass {
			continue
		}
		reply.Images = append(reply.Images, &pb.ImageInfo{
			Name: i.Name + ":" + i.Version,
			Repo: i.Name,
			Tag:  i.Version,
			Size: i.FileSize,
		})
	}

	return &reply, nil
}

func (s *ImageServer) ListDB(ctx context.Context, in *pb.ListDBRequest) (*pb.ListDBReply, error) {
	images, err := db.ListImages()
	if err != nil {
		log.Warnf("Failed to get images list: %v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListDBReply
	for _, i := range images {
		reply.Images = append(reply.Images, &pb.ImageDBInfo{
			Id:             i.ID,
			Name:           i.Name,
			Version:        i.Version,
			Description:    i.Description,
			VerifyStatus:   i.VerifyStatus,
			ApprovalStatus: i.ApprovalStatus,
			Size:           i.FileSize,
			UpdateAt:       i.UpdatedAt,
		})
	}

	return &reply, nil
}

func (s *ImageServer) Upload(stream pb.Image_UploadServer) error {
	// 镜像大小 版本 描述 签名(如果有)等信息在第一个数据包
	req, err := stream.Recv()
	if err != nil {
		log.Errorf("Receive image metadata, err=%v", err)
		return status.Error(codes.InvalidArgument, "读取参数失败")
	}

	if err := s.uploadCheckMetadata(req.Info); err != nil {
		return err
	}

	i, err := db.QueryImageByNameVersion(req.Info.Name, req.Info.Version)
	if err != nil {
		log.Errorf("Check duplicate image, err=%v", err)
		return rpc.ErrDatabaseFail
	} else if i != nil {
		return status.Error(codes.InvalidArgument, "镜像版本已存在")
	}

	ss, err := db.GetSecuritySwitch()
	if err != nil {
		log.Errorf("db.GetSecuritySwitch err=%v", err)
		return rpc.ErrInternal
	}

	img := db.ImageInfo{
		Name:        req.Info.Name,
		Version:     req.Info.Version,
		Description: req.Info.Description,
		FileSize:    req.Info.Size,
		CheckSum:    req.Info.Checksum,
	}

	f, err := s.receiveImageFile(ss, &img, req.Sign, ImageUploadStream{stream})
	if err != nil {
		return err
	}

	id, err := db.CreateImage(&img)
	if err != nil {
		log.Errorf("db.CreateImage data=%+v err=%v", img, err)
		f.Destroy()
		return rpc.ErrInternal
	}

	s.noticeNewImage(&img)
	stream.SendAndClose(&pb.UploadReply{ImageId: id})
	return nil
}

// 更新镜像文件 文件签名信息 描述信息
func (s *ImageServer) Update(stream pb.Image_UpdateServer) error {
	req, err := stream.Recv()
	if err != nil {
		log.Errorf("Receive image metadata, err=%v", err)
		return status.Error(codes.InvalidArgument, "读取参数失败")
	}

	if err := s.updateCheckMetadata(req.Info); err != nil {
		return err
	}

	img, err := db.QueryImageByID(req.ImageId)
	if err != nil {
		log.Errorf("Query image id=%v err=%v", req.ImageId, err)
		return rpc.ErrDatabaseFail
	} else if img == nil {
		return status.Error(codes.InvalidArgument, "镜像不存在")
	}

	img.Description = req.Info.Description
	if req.Info.Size <= 0 {
		if err = db.UpdateImage(img); err != nil {
			log.Errorf("db.UpdateImage data=%+v err=%v", img, err)
			return rpc.ErrDatabaseFail
		}
		stream.SendAndClose(&pb.UpdateReply{})
		return nil
	}

	// update image file
	ss, err := db.GetSecuritySwitch()
	if err != nil {
		log.Errorf("Get security switch status, err=%v", err)
		return rpc.ErrInternal
	}
	if ss && img.ApprovalStatus == db.ApprovalPass {
		return status.Error(codes.InvalidArgument, "镜像已审批，禁止更新镜像文件")
	}

	oldFile := img.FilePath
	img.FileSize = req.Info.Size
	img.CheckSum = req.Info.Checksum

	f, err := s.receiveImageFile(ss, img, req.Sign, ImageUpdateStream{stream})
	if err != nil {
		return err
	}

	if err := db.UpdateImage(img); err != nil {
		log.Errorf("db.UpdateImage data=%+v err=%v", img, err)
		f.Destroy()
		return rpc.ErrDatabaseFail
	}

	os.Remove(oldFile) // remove old image file
	s.noticeNewImage(img)
	stream.SendAndClose(&pb.UpdateReply{})
	return nil
}

func (s *ImageServer) Download(in *pb.DownloadRequest, stream pb.Image_DownloadServer) error {
	img, err := db.QueryImageByID(in.ImageId)
	if err != nil {
		return rpc.ErrInternal
	} else if img == nil {
		return rpc.ErrNotFound
	}

	f, err := os.Open(img.FilePath)
	if err != nil {
		log.Errorf("Open file=%v err=%v", img.FilePath, err)
		return rpc.ErrInternal
	}
	defer f.Close()

	/* 获取扩展名, 否则下载的文件无扩展名 */
	trimmed := strings.TrimSpace(img.FilePath)
	ext := filepath.Ext(trimmed)

	reply := pb.DownloadReply{
		Info: &pb.UploadInfo{
			Name:     img.Name,
			Version:  img.Version,
			Type:	  ext,
			Checksum: img.CheckSum,
			Size:     img.FileSize,
		},
	}

	if err := stream.Send(&reply); err != nil {
		log.Errorf("Send image metadata, err=%v", err)
		return rpc.ErrInternal
	}

	r := bufio.NewReader(f)
	buf := make([]byte, 64*1024)
	for {
		n, err := r.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Infof("Read image file=%+v err=%v", f, err)
			return rpc.ErrInternal
		}

		if err = stream.Send(&pb.DownloadReply{ChunkData: buf[:n]}); err != nil {
			log.Infof("Send image file chunk err=%v", err)
			return rpc.ErrInternal
		}
	}

	return nil
}

func (s *ImageServer) Approve(ctx context.Context, in *pb.ApproveRequest) (*pb.ApproveReply, error) {
	i, err := db.QueryImageByID(in.ImageId)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, status.Errorf(codes.NotFound, "镜像不存在")
		}
		return nil, rpc.ErrDatabaseFail
	}

	if in.Approve {
		if i.VerifyStatus != db.VerifyPass {
			return nil, status.Errorf(codes.FailedPrecondition, "镜像校验未通过，无法审批通过")
		}
		i.ApprovalStatus = db.ApprovalPass
	} else {
		if in.RejectReason == "" && i.VerifyStatus == db.VerifyPass {
			return nil, status.Errorf(codes.InvalidArgument, "参数错误：拒绝原因不能为空")
		}
		i.ApprovalStatus = db.ApprovalReject
	}
	i.RejectReason = in.RejectReason

	if err := db.UpdateImage(i); err != nil {
		log.Infof("Approve image=%+v err=%v", i, err)
		return nil, rpc.ErrDatabaseFail
	}

	s.noticeNewImage(i)
	return &pb.ApproveReply{}, nil
}

func (s *ImageServer) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveReply, error) {
	images, err := db.QueryImageByIDs(in.ImageIds)
	if err != nil {
		log.Infof("DB query image info err=%v", err)
		return nil, rpc.ErrDatabaseFail
	}

	imgs, err := db.RemoveImages(in.ImageIds)
	if err != nil {
		return nil, rpc.ErrDatabaseFail
	}

	for _, i := range imgs {
		if len(i.FilePath) > 0 {
			err := os.Remove(i.FilePath)
			log.Infof("Remove image file path=%v err=%v", i.FilePath, err)
		}
	}

	go func() {
		var toRemove []string
		for _, i := range images {
			toRemove = append(toRemove, i.ImageId)
		}
		s.noticeAgentSync(toRemove, nil)
	}()

	return &pb.RemoveReply{}, nil
}
