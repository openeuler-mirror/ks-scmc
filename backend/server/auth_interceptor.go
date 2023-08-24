package server

import (
	"context"
	"strings"
	"time"

	"scmc/model"
	"scmc/rpc"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const AuthKeySeprator = ':'

// AuthInterceptor server interceptor for authentication and authorization
type AuthInterceptor struct {
}

// NewAuthInterceptor returns a new auth interceptor
func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{}
}

// Unary returns server interceptor for unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
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

			reqDump := ""
			if r, ok := req.(proto.Message); ok {
				reqDump = proto.MarshalTextString(r)
			}

			repDump := ""
			if r, ok := reply.(proto.Message); ok {
				repDump = proto.MarshalTextString(r)
			}

			logger("%s %s\nREQUEST: %sREPLY: %sERR: %v", addr, info.FullMethod, reqDump, repDump, err)
			log.Infof("%s %s COST: %v ms", addr, info.FullMethod, time.Since(ts).Milliseconds())

			// TODO append runtime logs
			// AppendRuntimeLog(info.FullMethod, req, err)
		}()

		e := interceptor.check(ctx, info.FullMethod)
		if e != nil {
			log.Infof("AuthInterceptor error: %v", e)
			return nil, e
		} else {
			reply, err = handler(ctx, req)
			return reply, err
		}
	}
}

func (interceptor *AuthInterceptor) check(ctx context.Context, method string) error {
	if method == "/user.User/Login" || method == "/user.User/Signup" {
		return nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Infof("get metadata from incoming context failed")
		return rpc.ErrUnauthenticated
	}

	values := md["authorization"]
	if len(values) == 0 {
		log.Infof("'authorization' not exist in request metadata.")
		return rpc.ErrUnauthenticated
	}

	authorization := values[0]
	i := strings.IndexRune(authorization, AuthKeySeprator)
	if i == -1 {
		log.Infof("invalid authorization metadata: %v", authorization)
		return rpc.ErrUnauthenticated
	}

	userID, accessToken := authorization[:i], authorization[i+1:]

	ok, err := model.CheckUserSession(userID, accessToken)
	if err != nil {
		log.Info("model.CheckUserSession error")
		return rpc.ErrInternal
	} else if !ok {
		log.Info("invalid user session")
		return rpc.ErrUnauthenticated
	}

	return nil
}
