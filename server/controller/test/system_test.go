package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/system"
)

func TestSub(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSystemClient(conn)

		c, err := cli.Subscribe(ctx, &pb.SubscribeRequest{})
		if err != nil {
			t.Fatalf("Subscribe err=%v", err)
		}
		for {
			m, err := c.Recv()
			if err != nil {
				t.Fatalf("Subscribe recv err=%v", err)
			}
			t.Logf("Subscribe msg=%+v", m)
		}
	})
}

func TestGetSecuritySwitch(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSystemClient(conn)
		request := pb.GetSecuritySwitchRequest{}

		reply, err := cli.GetSecuritySwitch(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("reply: %+v", reply)
	})
}

func TestSetSecuritySwitch(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewSystemClient(conn)
		request := pb.SetSecuritySwitchRequest{
			IsOn: true,
		}

		reply, err := cli.SetSecuritySwitch(ctx, &request)
		if err != nil {
			t.Errorf("List: %v", err)
		}

		t.Logf("reply: %+v", reply)
	})
}
