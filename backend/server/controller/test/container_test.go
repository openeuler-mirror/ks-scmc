package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/container"
)

func TestContainerList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.ListRequest{
			Header:  &common.RequestHeader{},
			NodeIds: []int64{1},
			ListAll: true,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %v", reply)
	})
}

func TestContainerCreate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.CreateRequest{
			Header: &common.RequestHeader{},
			NodeId: 1,
			Name:   "redis-data",
			Config: &pb.CreateContainerConfig{
				Hostname:   "test_hostname",
				DomainName: "test_domain",
				Image:      "redis",
			},
			// HostConfig: &pb.HostConfig{},
		}

		reply, err := cli.Create(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %v", reply)
	})
}
