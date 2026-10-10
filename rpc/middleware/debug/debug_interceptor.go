// package debug provides grpc server interceptor to logging rpc request/reply data.
package debug

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
)

func rpcNeedDebug(method string) bool {
	switch method {
	case "/user.User/Login",
		"/user.User/Signup",
		"/user.User/UpdatePassword",
		"/user.User/CreateUser",
		"/user.User/UpdateUser":
		return false
	}
	return true
}

// Unary returns a server interceptor function to logging unary RPC
func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		ts := time.Now()

		var addr string
		if p, ok := peer.FromContext(ctx); ok {
			addr = p.Addr.String()
		}

		if !rpcNeedDebug(info.FullMethod) {
			goto handle
		}

		log.Debugf("%s %s\nREQUEST: %+v", addr, info.FullMethod, req)
		defer func() {
			duration := time.Since(ts)
			if err != nil {
				log.Infof("%s %s\nREQUEST: %+v ERR: %v", addr, info.FullMethod, req, err)
			} else {
				if duration.Seconds() > 0.999 {
					log.Infof("%s %s COST: %v", addr, info.FullMethod, duration)
				} else {
					log.Debugf("%s %s\nREQUEST: %+v\nREPLY: %+v\nCOST: %v", addr, info.FullMethod, req, reply, duration)
				}
			}
		}()

	handle:
		reply, err = handler(ctx, req)
		return reply, err
	}
}

func StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// wrapped := WrapServerStream(stream)
		// wrapped.WrappedContext = stream.Context()

		ts := time.Now()
		var addr string
		if p, ok := peer.FromContext(stream.Context()); ok {
			addr = p.Addr.String()
		}

		err := handler(srv, stream)

		duration := time.Since(ts)
		if err != nil {
			log.Infof("%s RPC=%v COST=%v, ERR=%v", addr, info.FullMethod, duration, err)
		} else {
			log.Infof("%s RPC=%v COST=%v", addr, info.FullMethod, duration)
		}
		return err
	}
}
