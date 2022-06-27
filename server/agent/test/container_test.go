package test

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/container"
)

func TestContainerList(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.ListRequest{
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
			ContainerId: "030f6ec2fede",
		}

		reply, err := cli.Inspect(ctx, &request)
		if err != nil {
			t.Errorf("Inspect: %v", err)
		}

		t.Logf("Inspect reply: %+v", reply)
	})
}

func TestMonitorHistory(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewContainerClient(conn)
		request := pb.MonitorHistoryRequest{
			StartTime:   time.Now().Unix() - 3600,
			EndTime:     time.Now().Unix(),
			Interval:    1,
			ContainerId: "cadvisor",
		}

		reply, err := cli.MonitorHistory(ctx, &request)
		if err != nil {
			t.Errorf("MonitorHistory: %v", err)
		} else {
			t.Logf("MonitorHistory %+v", reply.CpuUsage)
			t.Logf("MonitorHistory %+v", reply.MemoryUsage)
			t.Logf("MonitorHistory %+v", reply.DiskUsage)
			t.Logf("MonitorHistory %+v", reply.NetRx)
			t.Logf("MonitorHistory %+v", reply.NetTx)
		}
	})
}
