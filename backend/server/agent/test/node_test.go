package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/node"
)

func TestNodeStatus(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.StatusRequest{
			Header:  &common.RequestHeader{},
			NodeIds: []int64{},
		}

		reply, err := cli.Status(ctx, &request)
		if err != nil {
			t.Errorf("Status: %v", err)
		}

		t.Logf("Status reply: %v", reply)
	})
}
