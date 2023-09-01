package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/node"
)

func TestNodeList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.ListRequest{}

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
			Ids: []int64{1},
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
			NodeIds: []int64{1},
		}

		reply, err := cli.Status(ctx, &request)
		if err != nil {
			t.Errorf("Status: %v", err)
		}

		t.Logf("Status reply: %v", reply)
	})
}

func TestUpdateNode(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNodeClient(conn)
		request := pb.UpdateRequest{
			NodeId:  1,
			Comment: "12345678",
			RscLimit: &pb.ResourceLimit{
				CpuLimit:    1.0,
				MemoryLimit: 2048,
			},
		}

		reply, err := cli.Update(ctx, &request)
		if err != nil {
			t.Errorf("UpdateNode: %v", err)
		} else {
			t.Logf("UpdateNode reply: %v", reply)
		}
	})
}
