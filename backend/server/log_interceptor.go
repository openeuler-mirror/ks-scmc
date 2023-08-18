package server

import (
	"context"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	// "google.golang.org/protobuf/proto"
)

// LogInterceptor is a server interceptor for request and reply logging
type LogInterceptor struct{}

// NewLogInterceptor returns a new log interceptor
func NewLogInterceptor() *LogInterceptor {
	return &LogInterceptor{}
}

// Unary returns a server interceptor function to logging unary RPC
func (interceptor *LogInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		var addr string
		if p, ok := peer.FromContext(ctx); ok {
			addr = p.Addr.String()
		}

		ts := time.Now()
		defer func() {
			logger := log.Debugf
			if err != nil {
				logger = log.Infof
			}
			logger("%s %s\nREQUEST: %vREPLY: %vERR: %v", addr, info.FullMethod,
				proto.MarshalTextString(req.(proto.Message)), proto.MarshalTextString(reply.(proto.Message)), err)
			log.Infof("%s %s COST: %v ms", addr, info.FullMethod, time.Since(ts).Milliseconds())
		}()

		reply, err = handler(ctx, req)
		return reply, err
	}
}
