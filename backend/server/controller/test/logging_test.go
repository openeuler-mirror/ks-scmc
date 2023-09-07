package test

import (
	"context"
	pb "scmc/rpc/pb/logging"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestListRuntimeLog(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewLoggingClient(conn)
		request := pb.ListRuntimeRequest{
			// NodeId:    1,
			StartTime: time.Now().AddDate(0, 0, -2).Unix(),
			EndTime:   time.Now().Unix(),
			PageNo:    4,
		}

		reply, err := cli.ListRuntime(ctx, &request)
		if err != nil {
			t.Errorf("ListRuntime: %v", err)
		} else {
			for _, l := range reply.Logs {
				t.Logf("%v", l)
			}
			t.Logf("PageNo=%v PageSize=%v TotalPages=%v", reply.PageNo, reply.PageSize, reply.TotalPages)
		}
	})
}

func TestListWarnLog(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewLoggingClient(conn)
		request := pb.ListWarnRequest{
			// NodeId:    1,
			PageNo: 0,
		}

		reply, err := cli.ListWarn(ctx, &request)
		if err != nil {
			t.Errorf("ListWarn: %v", err)
		} else {
			for _, w := range reply.Logs {
				t.Logf("%v", w)
			}
			t.Logf("PageNo=%v PageSize=%v TotalPages=%v", reply.PageNo, reply.PageSize, reply.TotalPages)
		}
	})
}
