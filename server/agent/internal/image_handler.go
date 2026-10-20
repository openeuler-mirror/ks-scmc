package internal

import (
	"archive/tar"
	"context"
	"os"
	"io"
	"strings"
	"sync"
	"regexp"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"scmc/model/docker"
	"scmc/model/registry"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
)

var imageSyncGuard sync.Mutex

type ImageServer struct {
	pb.UnimplementedImageServer
}

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	cli, err := docker.Client()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warnf("ImageList: %v", err)
		return nil, transDockerError(err)
	}

	var reply pb.ListReply
	for _, i := range images {
		for _, s := range i.RepoTags {
			n := strings.IndexRune(s, ':')
			if n > -1 {
				reply.Images = append(reply.Images, &pb.ImageInfo{
					Name: s,
					Repo: s[0:n],
					Tag:  s[n+1:],
				})
			} else {
				reply.Images = append(reply.Images, &pb.ImageInfo{
					Name: s,
					Repo: s,
				})
			}
		}
	}
	return &reply, nil
}

func (s *ImageServer) AgentSync(ctx context.Context, in *pb.AgentSyncRequest) (*pb.AgentSyncReply, error) {
	imageSyncGuard.Lock()
	defer imageSyncGuard.Unlock()

	if len(in.ToRemove) == 0 && len(in.ToLoad) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	cli, err := docker.Client()
	if err != nil {
		log.Error("acquire docker client: ", err)
		return nil, rpc.ErrInternal
	}

	bg := context.Background()
	opt := types.ImageRemoveOptions{Force: true}
	for _, repoTag := range in.ToRemove {
		_, err = cli.ImageRemove(bg, repoTag, opt)
		if err != nil {
			log.Infof("remove image '%s': %v", repoTag, err)
		} else {
			log.Infof("remove image '%s' OK", repoTag)
		}
	}

	loadImage := func(path, id, tag string) {
		f, err := os.Open(path)
		if err != nil {
			log.Errorf("open image file '%s': %v", path, err)
			return
		}
		defer f.Close()

		isTarFileSystem := true
		reader := tar.NewReader(f)
		for {
			header, err := reader.Next()
			if err != nil {
				if err.Error() == "EOF" {
					break
				}
				log.Errorf("无法读取归档文件：", err)
				return
			}
			log.Debugln(header.Name)
			if header.Name == registry.ManifestFileName {
				isTarFileSystem = false
				break
			}
		}
		
		if isTarFileSystem {
			tarf, err := os.Open(path)
			if err != nil {
				log.Errorf("open image file '%s': %v", path, err)
				return
			}
			defer f.Close()

			resp, err := cli.ImageImport(bg, types.ImageImportSource{
				Source:     tarf,
				SourceName: "-", // 从标准输入流读取
			}, "", types.ImageImportOptions{})
			if err != nil {
				log.Errorf("load tar image file '%s': %v", path, err)
			}
			defer resp.Close()
			
			var respString string
			for {
				data := make([]byte, 4096)
				n, err := resp.Read(data)
				if err == io.EOF {
					break
				}
				respString += string(data[:n])
			}
			log.Infof("Imported image ID: %s ", respString)
			re := regexp.MustCompile(`sha256:([a-f0-9]+)`)
			match := re.FindStringSubmatch(respString)
			if len(match) > 1 {
				id = match[1]
			}
			log.Infof("Imported image ID: %s ", id)
		} else {
			imgf, err := os.Open(path)
			if err != nil {
				log.Errorf("open image file '%s': %v", path, err)
				return
			}
			defer f.Close()
			if _, err := cli.ImageLoad(bg, imgf, true); err != nil {
				log.Errorf("load image file '%s': %v", path, err)
				return
			}
		}

		if err := cli.ImageTag(bg, id, tag); err != nil {
			log.Errorf("tag image '%s' to '%s': %v", id, tag, err)
		}
		log.Infof("load image file=%s id=%s tag=%s OK", path, id, tag)
	}

	for _, i := range in.ToLoad {
		loadImage(i.Path, i.ImageId, i.Tag)
	}

	return &pb.AgentSyncReply{}, nil
}
