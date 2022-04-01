package internal

import (
	"context"

	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
)

type ImageServer struct {
	pb.UnimplementedImageServer
}

/*
func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := dockerCli()
	if err != nil {
		return nil, rpc.ErrInternal
	}

	list, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		log.Warnf("ImageList: %v", err)
		return nil, transDockerError(err)
	}

	for _, image := range list {
		for _, s := range image.RepoTags {
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
*/

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	images, err := model.ListImages()
	if err != nil {
		log.Warnf("Failed to get images list: %v", err)
		return nil, rpc.ErrInternal
	}

	reply := pb.ListReply{}
	for _, image := range images {
		// if image.VerifyStatus != model.VerifyPass || image.ApprovalStatus != model.ApprovalPass {
		if image.VerifyStatus != model.VerifyPass {
			continue
		}
		reply.Images = append(reply.Images, &pb.ImageInfo{
			Name: image.Name + ":" + image.Version,
			Repo: image.Name,
			Tag:  image.Version,
		})
	}

	return &reply, nil
}
