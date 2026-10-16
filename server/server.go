// package server implement rpc servers
package server

import (
	"context"
	"time"

	pb "scmc/rpc/pb/container"

	"google.golang.org/grpc"
)

func ListBackupImage(conn *grpc.ClientConn) (map[string]bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	cli := pb.NewContainerClient(conn)
	reply, err := cli.ListBackupImage(ctx, &pb.ListBackupImageRequest{})
	if err != nil {
		return nil, err
	}

	r := make(map[string]bool, len(reply.Images))
	for _, i := range reply.Images {
		r[i] = true
	}

	return r, nil
}
