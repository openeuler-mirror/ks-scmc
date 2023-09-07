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

func TestNetworkIPtablesEnable(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.EnableIPtablesRequest{
			NodeId:      1,
			Enable:      true,
			ContainerId: "c3",
		}
		reply, err := cli.EnableIPtables(ctx, &request)
		if err != nil {
			t.Errorf("EnableIPtables: %v", err)
		}

		t.Logf("EnableIPtables reply: %+v", reply)
	})
}

func TestNetworkIPtablesList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ListIPtablesRequest{
			NodeId:      1,
			ContainerId: "c3",
		}
		reply, err := cli.ListIPtables(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		for _, e := range reply.ChainRules {
			for _, r := range e.Rule {
				t.Logf("[%v]:[%+v]", e.Chain, r)
			}
		}
	})
}

func TestNetworkIPtablesCreate(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.CreateIPtablesRequest{
			NodeId:      1,
			ContainerId: "c3",
			Rule: &pb.RuleInfo{
				Source:      "192.168.122.200/24",
				Destination: "192.168.122.250",
				Protocol:    "tcp",
				SrcPort:     "25",
				DestPort:    "22",
			},
		}
		reply, err := cli.CreateIPtables(ctx, &request)
		if err != nil {
			t.Errorf("Create: %v", err)
		}

		t.Logf("Create reply: %+v", reply)
	})
}

func TestNetworkIPtablesModify(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.ModifyIPtablesRequest{
			NodeId:      1,
			ContainerId: "c3",
			OldRule: &pb.RuleInfo{
				Source:      "192.168.122.200/24",
				Destination: "192.168.122.250",
				Protocol:    "tcp",
				SrcPort:     "25",
				DestPort:    "22",
			},
			NewRule: &pb.RuleInfo{
				Source:      "192.168.122.50",
				Destination: "192.168.122.150",
				Protocol:    "udp",
				SrcPort:     "22",
				DestPort:    "23",
			},
		}
		reply, err := cli.ModifyIPtables(ctx, &request)
		if err != nil {
			t.Errorf("Modify: %v", err)
		}

		t.Logf("Modify reply: %+v", reply)
	})
}

func TestNetworkIPtablesRemove(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewNetworkClient(conn)
		request := pb.RemoveIPtablesRequest{
			NodeId:      1,
			ContainerId: "c3",
			Rule: &pb.RuleInfo{
				Source:      "192.168.122.50",
				Destination: "192.168.122.150",
				Protocol:    "udp",
				SrcPort:     "22",
				DestPort:    "23",
			},
		}
		reply, err := cli.RemoveIPtables(ctx, &request)
		if err != nil {
			t.Errorf("Remove: %v", err)
		}

		t.Logf("Remove reply: %+v", reply)
	})
}
