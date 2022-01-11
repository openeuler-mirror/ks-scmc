package common

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// debug level logging: request and reply
// info level logging: time cost
func basicUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		ts := time.Now()
		log.Debugf("RPC%s request: %v", info.FullMethod, req)
		defer func() {
			log.Debugf("RPC%s reply: %v", info.FullMethod, reply)
			log.Infof("RPC%s cost: %v ms", info.FullMethod, time.Since(ts).Milliseconds())
		}()

		reply, err = handler(ctx, req)
		return reply, err
	}
}

func UnaryServerInterceptor() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(basicUnaryServerInterceptor()),
	}
}
