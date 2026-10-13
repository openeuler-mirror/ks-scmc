package internal

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"scmc/model/docker"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
)

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
	if len(in.ToRemove) == 0 && len(in.ToPull) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	go func() {
		if err := docker.RemoveImage(true, in.ToRemove...); err != nil {
			log.Infof("remove images=%v err=%v", in.ToRemove, err)
		}

		for _, i := range in.ToPull {
			if err := docker.PullImage(i); err != nil {
				log.Infof("pull image=%v err=%v", i, err)
			}
		}
	}()

	return &pb.AgentSyncReply{}, nil
}
