package internal

import (
	"context"
	"strings"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/image"
)

type ImageServer struct {
	pb.UnimplementedImageServer
}

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	reply := pb.ListReply{}

	cli, err := model.DockerClient()
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

func (s *ImageServer) agentSync(toRemove, toPull []string) {
	cli, err := model.DockerClient()
	if err != nil {
		log.Infof("agentSync get docker client err=%v", err)
		goto stagePull
	}

	for _, i := range toRemove {
		if _, err := cli.ImageRemove(context.Background(), i, types.ImageRemoveOptions{}); err != nil {
			log.Infof("agentSync remove image=%v err=%v", i, err)
		}
	}

stagePull:
	for _, i := range toPull {
		if err := ensureImage(cli, i); err != nil {
			log.Infof("agentSync pull image=%v err=%v", i, err)
		}
	}
}

func (s *ImageServer) AgentSync(ctx context.Context, in *pb.AgentSyncRequest) (*pb.AgentSyncReply, error) {
	if len(in.ToRemove) == 0 && len(in.ToPull) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	go s.agentSync(in.ToRemove, in.ToPull)

	return &pb.AgentSyncReply{}, nil
}
