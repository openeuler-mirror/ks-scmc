package middleware

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ServerStreamWrapper struct {
	ss  grpc.ServerStream
	ctx context.Context
}

func NewServerStreamWrapper(ss grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return ServerStreamWrapper{
		ss:  ss,
		ctx: ctx,
	}
}

func (w ServerStreamWrapper) Context() context.Context {
	return w.ctx
}

func (w ServerStreamWrapper) RecvMsg(msg interface{}) error {
	return w.ss.RecvMsg(msg)
}

func (w ServerStreamWrapper) SendMsg(msg interface{}) error {
	return w.ss.SendMsg(msg)
}

func (w ServerStreamWrapper) SendHeader(md metadata.MD) error {
	return w.ss.SendHeader(md)
}

func (w ServerStreamWrapper) SetHeader(md metadata.MD) error {
	return w.ss.SetHeader(md)
}

func (w ServerStreamWrapper) SetTrailer(md metadata.MD) {
	w.ss.SetTrailer(md)
}
