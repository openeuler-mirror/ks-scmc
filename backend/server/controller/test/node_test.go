package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/node"
)

func TestNodeList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.ListRequest{
			Header: &common.RequestHeader{},
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %v", reply)
	})
}

func TestNodeCreate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.CreateRequest{
			Header:  &common.RequestHeader{},
			Name:    "test",
			Address: "10.200.12.181",
			Comment: "123",
		}

		reply, err := cli.Create(ctx, &request)
		if err != nil {
			t.Errorf("Create: %v", err)
		}

		t.Logf("Create reply: %v", reply)
	})
}

func TestNodeRemove(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.RemoveRequest{
			Header: &common.RequestHeader{},
			Ids:    []int64{1},
		}

		reply, err := cli.Remove(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %v", reply)
	})
}

func TestNodeStatus(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.StatusRequest{
			Header:  &common.RequestHeader{},
			NodeIds: []int64{1},
		}

		reply, err := cli.Status(ctx, &request)
		if err != nil {
			t.Errorf("Status: %v", err)
		}

		t.Logf("Status reply: %v", reply)
	})
}
