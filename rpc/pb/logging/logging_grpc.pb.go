// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package logging

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// LoggingClient is the client API for Logging service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type LoggingClient interface {
	// 查询审计事件日志列表
	ListRuntime(ctx context.Context, in *ListRuntimeRequest, opts ...grpc.CallOption) (*ListRuntimeReply, error)
	// 查询告警事件列表
	ListWarn(ctx context.Context, in *ListWarnRequest, opts ...grpc.CallOption) (*ListWarnReply, error)
	// 设置告警事件已读
	ReadWarn(ctx context.Context, in *ReadWarnRequest, opts ...grpc.CallOption) (*ReadWarnReply, error)
}

type loggingClient struct {
	cc grpc.ClientConnInterface
}

func NewLoggingClient(cc grpc.ClientConnInterface) LoggingClient {
	return &loggingClient{cc}
}

func (c *loggingClient) ListRuntime(ctx context.Context, in *ListRuntimeRequest, opts ...grpc.CallOption) (*ListRuntimeReply, error) {
	out := new(ListRuntimeReply)
	err := c.cc.Invoke(ctx, "/logging.Logging/ListRuntime", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loggingClient) ListWarn(ctx context.Context, in *ListWarnRequest, opts ...grpc.CallOption) (*ListWarnReply, error) {
	out := new(ListWarnReply)
	err := c.cc.Invoke(ctx, "/logging.Logging/ListWarn", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *loggingClient) ReadWarn(ctx context.Context, in *ReadWarnRequest, opts ...grpc.CallOption) (*ReadWarnReply, error) {
	out := new(ReadWarnReply)
	err := c.cc.Invoke(ctx, "/logging.Logging/ReadWarn", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// LoggingServer is the server API for Logging service.
// All implementations must embed UnimplementedLoggingServer
// for forward compatibility
type LoggingServer interface {
	// 查询审计事件日志列表
	ListRuntime(context.Context, *ListRuntimeRequest) (*ListRuntimeReply, error)
	// 查询告警事件列表
	ListWarn(context.Context, *ListWarnRequest) (*ListWarnReply, error)
	// 设置告警事件已读
	ReadWarn(context.Context, *ReadWarnRequest) (*ReadWarnReply, error)
	mustEmbedUnimplementedLoggingServer()
}

// UnimplementedLoggingServer must be embedded to have forward compatible implementations.
type UnimplementedLoggingServer struct {
}

func (UnimplementedLoggingServer) ListRuntime(context.Context, *ListRuntimeRequest) (*ListRuntimeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRuntime not implemented")
}
func (UnimplementedLoggingServer) ListWarn(context.Context, *ListWarnRequest) (*ListWarnReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListWarn not implemented")
}
func (UnimplementedLoggingServer) ReadWarn(context.Context, *ReadWarnRequest) (*ReadWarnReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReadWarn not implemented")
}
func (UnimplementedLoggingServer) mustEmbedUnimplementedLoggingServer() {}

// UnsafeLoggingServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to LoggingServer will
// result in compilation errors.
type UnsafeLoggingServer interface {
	mustEmbedUnimplementedLoggingServer()
}

func RegisterLoggingServer(s grpc.ServiceRegistrar, srv LoggingServer) {
	s.RegisterService(&Logging_ServiceDesc, srv)
}

func _Logging_ListRuntime_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRuntimeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoggingServer).ListRuntime(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logging.Logging/ListRuntime",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoggingServer).ListRuntime(ctx, req.(*ListRuntimeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Logging_ListWarn_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListWarnRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoggingServer).ListWarn(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logging.Logging/ListWarn",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoggingServer).ListWarn(ctx, req.(*ListWarnRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Logging_ReadWarn_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReadWarnRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(LoggingServer).ReadWarn(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/logging.Logging/ReadWarn",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(LoggingServer).ReadWarn(ctx, req.(*ReadWarnRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Logging_ServiceDesc is the grpc.ServiceDesc for Logging service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Logging_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "logging.Logging",
	HandlerType: (*LoggingServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ListRuntime",
			Handler:    _Logging_ListRuntime_Handler,
		},
		{
			MethodName: "ListWarn",
			Handler:    _Logging_ListWarn_Handler,
		},
		{
			MethodName: "ReadWarn",
			Handler:    _Logging_ReadWarn_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "logging.proto",
}
