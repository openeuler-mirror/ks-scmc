package internal

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/image"
)

type ImageServer struct {
	pb.UnimplementedImageServer
}

func (s *ImageServer) List(ctx context.Context, in *pb.ListRequest) (*pb.ListReply, error) {
	if in.NodeId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	nodeInfo, err := model.QueryNodeByID(in.NodeId)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	conn, err := getAgentConn(nodeInfo.Address)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	cli := pb.NewImageClient(conn)
	ctx_, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	reply, err := cli.List(ctx_, in)
	if err != nil {
		log.Warnf("image list: %v", err)
		return nil, rpc.ErrInternal
	}

	return reply, nil
}
