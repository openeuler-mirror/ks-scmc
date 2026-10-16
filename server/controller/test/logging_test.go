package test

import (
	"context"
	"io"
	pb "scmc/rpc/pb/logging"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestSubscribeWarn(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewLoggingClient(conn)
		request := pb.SubscribeWarnRequest{}

		stream, err := cli.SubscribeWarn(ctx, &request)
		if err != nil {
			t.Errorf("download: %v", err)
		}

		for {
			if stream.Context().Err() == context.Canceled || stream.Context().Err() == context.DeadlineExceeded {
				t.Errorf("context error: %v", stream.Context().Err())
			}

			reply, err := stream.Recv()
			if err == io.EOF {
				t.Logf("no more data")
				break
			}
			if err != nil {
				t.Errorf("cannot receive chunk data: %v", err)
				break
			}
			for _, log := range reply.Logs {
				t.Logf("log: %++v", log)
			}

		}
	})
}

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

func TestReadWarn(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewLoggingClient(conn)
		request := pb.ReadWarnRequest{
			Ids: []int64{1},
		}

		_, err := cli.ReadWarn(ctx, &request)
		if err != nil {
			t.Errorf("ReadWarn: %v", err)
		}
	})
}
