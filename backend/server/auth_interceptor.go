package server

import (
	"context"
	"ksc-mcube/model"
	"ksc-mcube/rpc"
	"strings"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AuthInterceptor server interceptor for authentication and authorization
type AuthInterceptor struct {
}

// NewAuthInterceptor returns a new auth interceptor
func NewAuthInterceptor() *AuthInterceptor {
	return &AuthInterceptor{}
}

// Unary returns server interceptor for unary RPC
func (interceptor *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := interceptor.check(ctx, info.FullMethod)
		if err != nil {
			log.Infof("AuthInterceptor error: %v", err)
			return nil, err
		}

		log.Debug("--> unary interceptor: ", info.FullMethod)
		return handler(ctx, req)
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
	i := strings.IndexRune(authorization, ':')
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
