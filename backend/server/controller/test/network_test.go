package test

import (
	"context"
	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/network"
	"testing"

	"google.golang.org/grpc"
)

func TestNetworkList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ListRequest{
			Header:  &common.RequestHeader{},
			NodeIds: []int64{1},
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %+v", reply)
	})
}
