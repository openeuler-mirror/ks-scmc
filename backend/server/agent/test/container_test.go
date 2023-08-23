package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	common "scmc/rpc/pb/common"
	pb "scmc/rpc/pb/container"
)

func TestContainerList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.ListRequest{
			Header:  &common.RequestHeader{},
			ListAll: true,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, c := range reply.Containers {
			t.Logf("List container: %+v", c)
		}

	})
}

func TestContainerInspect1(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.InspectRequest{
			Header:      &common.RequestHeader{},
			ContainerId: "030f6ec2fede",
		}

		reply, err := cli.Inspect(ctx, &request)
		if err != nil {
			t.Errorf("Inspect: %v", err)
		}

		t.Logf("Inspect reply: %+v", reply)
	})
}
