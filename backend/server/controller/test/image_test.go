package test

import (
	"context"
	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/image"
	"testing"

	"google.golang.org/grpc"
)

func TestImageList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewImageClient(conn)
		request := pb.ListRequest{
			Header: &common.RequestHeader{},
			NodeId: 1,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, i := range reply.Images {
			t.Logf("%+v", i)
		}

	})
}
