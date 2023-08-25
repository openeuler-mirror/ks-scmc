package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/network"
)

func TestNetworkList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ListRequest{
			NodeId: 1,
		}

		reply, err := cli.List(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("List reply: %+v", reply)
	})
}

func TestNetworkConnect(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ConnectRequest{
			NodeId:      1,
			Interface:   "virbr2",
			ContainerId: "c3",
			IpAddress:   "172.60.100.55",
			IpMask:      "255.255.255.0",
			MacAddress:  "",
			Gateway:     "172.60.100.1",
		}

		reply, err := cli.Connect(ctx, &request)
		if err != nil {
			t.Errorf("Connect: %v", err)
		}

		t.Logf("Connect reply: %+v", reply)
	})
}

func TestNetworkDisconnect(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.DisconnectRequest{
			NodeId:      1,
			Interface:   "ovsbr0",
			ContainerId: "c3",
		}

		reply, err := cli.Disconnect(ctx, &request)
		if err != nil {
			t.Errorf("Disconnect: %v", err)
		}

		t.Logf("Disconnect reply: %+v", reply)
	})
}
