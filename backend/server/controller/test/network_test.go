package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/network"
)

func TestNetworkList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ListRequest{
			Header: &common.RequestHeader{},
			NodeId: 1,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %+v", reply)
	})
}
