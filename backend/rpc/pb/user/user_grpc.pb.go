// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package user

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

// UserClient is the client API for User service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type UserClient interface {
	Signup(ctx context.Context, in *SignupRequest, opts ...grpc.CallOption) (*SignupReply, error)
	Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginReply, error)
	Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutReply, error)
	UpdatePassword(ctx context.Context, in *UpdatePasswordRequest, opts ...grpc.CallOption) (*UpdatePasswordReply, error)
	// 管理员用户/角色管理
	ListUser(ctx context.Context, in *ListUserRequest, opts ...grpc.CallOption) (*ListUserReply, error)
	CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserReply, error)
	UpdateUser(ctx context.Context, in *UpdateUserRequest, opts ...grpc.CallOption) (*UpdateUserReply, error)
	RemoveUser(ctx context.Context, in *RemoveUserRequest, opts ...grpc.CallOption) (*RemoveUserReply, error)
	ListRole(ctx context.Context, in *ListRoleRequest, opts ...grpc.CallOption) (*ListRoleReply, error)
	CreateRole(ctx context.Context, in *CreateRoleRequest, opts ...grpc.CallOption) (*CreateRoleReply, error)
	UpdateRole(ctx context.Context, in *UpdateRoleRequest, opts ...grpc.CallOption) (*UpdateRoleReply, error)
	RemoveRole(ctx context.Context, in *RemoveRoleRequest, opts ...grpc.CallOption) (*RemoveRoleReply, error)
}

type userClient struct {
	cc grpc.ClientConnInterface
}

func NewUserClient(cc grpc.ClientConnInterface) UserClient {
	return &userClient{cc}
}

func (c *userClient) Signup(ctx context.Context, in *SignupRequest, opts ...grpc.CallOption) (*SignupReply, error) {
	out := new(SignupReply)
	err := c.cc.Invoke(ctx, "/user.User/Signup", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Login(ctx context.Context, in *LoginRequest, opts ...grpc.CallOption) (*LoginReply, error) {
	out := new(LoginReply)
	err := c.cc.Invoke(ctx, "/user.User/Login", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) Logout(ctx context.Context, in *LogoutRequest, opts ...grpc.CallOption) (*LogoutReply, error) {
	out := new(LogoutReply)
	err := c.cc.Invoke(ctx, "/user.User/Logout", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) UpdatePassword(ctx context.Context, in *UpdatePasswordRequest, opts ...grpc.CallOption) (*UpdatePasswordReply, error) {
	out := new(UpdatePasswordReply)
	err := c.cc.Invoke(ctx, "/user.User/UpdatePassword", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ListUser(ctx context.Context, in *ListUserRequest, opts ...grpc.CallOption) (*ListUserReply, error) {
	out := new(ListUserReply)
	err := c.cc.Invoke(ctx, "/user.User/ListUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) CreateUser(ctx context.Context, in *CreateUserRequest, opts ...grpc.CallOption) (*CreateUserReply, error) {
	out := new(CreateUserReply)
	err := c.cc.Invoke(ctx, "/user.User/CreateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) UpdateUser(ctx context.Context, in *UpdateUserRequest, opts ...grpc.CallOption) (*UpdateUserReply, error) {
	out := new(UpdateUserReply)
	err := c.cc.Invoke(ctx, "/user.User/UpdateUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) RemoveUser(ctx context.Context, in *RemoveUserRequest, opts ...grpc.CallOption) (*RemoveUserReply, error) {
	out := new(RemoveUserReply)
	err := c.cc.Invoke(ctx, "/user.User/RemoveUser", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) ListRole(ctx context.Context, in *ListRoleRequest, opts ...grpc.CallOption) (*ListRoleReply, error) {
	out := new(ListRoleReply)
	err := c.cc.Invoke(ctx, "/user.User/ListRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) CreateRole(ctx context.Context, in *CreateRoleRequest, opts ...grpc.CallOption) (*CreateRoleReply, error) {
	out := new(CreateRoleReply)
	err := c.cc.Invoke(ctx, "/user.User/CreateRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) UpdateRole(ctx context.Context, in *UpdateRoleRequest, opts ...grpc.CallOption) (*UpdateRoleReply, error) {
	out := new(UpdateRoleReply)
	err := c.cc.Invoke(ctx, "/user.User/UpdateRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userClient) RemoveRole(ctx context.Context, in *RemoveRoleRequest, opts ...grpc.CallOption) (*RemoveRoleReply, error) {
	out := new(RemoveRoleReply)
	err := c.cc.Invoke(ctx, "/user.User/RemoveRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserServer is the server API for User service.
// All implementations must embed UnimplementedUserServer
// for forward compatibility
type UserServer interface {
	Signup(context.Context, *SignupRequest) (*SignupReply, error)
	Login(context.Context, *LoginRequest) (*LoginReply, error)
	Logout(context.Context, *LogoutRequest) (*LogoutReply, error)
	UpdatePassword(context.Context, *UpdatePasswordRequest) (*UpdatePasswordReply, error)
	// 管理员用户/角色管理
	ListUser(context.Context, *ListUserRequest) (*ListUserReply, error)
	CreateUser(context.Context, *CreateUserRequest) (*CreateUserReply, error)
	UpdateUser(context.Context, *UpdateUserRequest) (*UpdateUserReply, error)
	RemoveUser(context.Context, *RemoveUserRequest) (*RemoveUserReply, error)
	ListRole(context.Context, *ListRoleRequest) (*ListRoleReply, error)
	CreateRole(context.Context, *CreateRoleRequest) (*CreateRoleReply, error)
	UpdateRole(context.Context, *UpdateRoleRequest) (*UpdateRoleReply, error)
	RemoveRole(context.Context, *RemoveRoleRequest) (*RemoveRoleReply, error)
	mustEmbedUnimplementedUserServer()
}

// UnimplementedUserServer must be embedded to have forward compatible implementations.
type UnimplementedUserServer struct {
}

func (UnimplementedUserServer) Signup(context.Context, *SignupRequest) (*SignupReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Signup not implemented")
}
func (UnimplementedUserServer) Login(context.Context, *LoginRequest) (*LoginReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Login not implemented")
}
func (UnimplementedUserServer) Logout(context.Context, *LogoutRequest) (*LogoutReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Logout not implemented")
}
func (UnimplementedUserServer) UpdatePassword(context.Context, *UpdatePasswordRequest) (*UpdatePasswordReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdatePassword not implemented")
}
func (UnimplementedUserServer) ListUser(context.Context, *ListUserRequest) (*ListUserReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListUser not implemented")
}
func (UnimplementedUserServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateUser not implemented")
}
func (UnimplementedUserServer) UpdateUser(context.Context, *UpdateUserRequest) (*UpdateUserReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateUser not implemented")
}
func (UnimplementedUserServer) RemoveUser(context.Context, *RemoveUserRequest) (*RemoveUserReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveUser not implemented")
}
func (UnimplementedUserServer) ListRole(context.Context, *ListRoleRequest) (*ListRoleReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListRole not implemented")
}
func (UnimplementedUserServer) CreateRole(context.Context, *CreateRoleRequest) (*CreateRoleReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateRole not implemented")
}
func (UnimplementedUserServer) UpdateRole(context.Context, *UpdateRoleRequest) (*UpdateRoleReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateRole not implemented")
}
func (UnimplementedUserServer) RemoveRole(context.Context, *RemoveRoleRequest) (*RemoveRoleReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveRole not implemented")
}
func (UnimplementedUserServer) mustEmbedUnimplementedUserServer() {}

// UnsafeUserServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to UserServer will
// result in compilation errors.
type UnsafeUserServer interface {
	mustEmbedUnimplementedUserServer()
}

func RegisterUserServer(s grpc.ServiceRegistrar, srv UserServer) {
	s.RegisterService(&User_ServiceDesc, srv)
}

func _User_Signup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Signup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/Signup",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Signup(ctx, req.(*SignupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Login_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LoginRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Login(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/Login",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Login(ctx, req.(*LoginRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_Logout_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(LogoutRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).Logout(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/Logout",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).Logout(ctx, req.(*LogoutRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_UpdatePassword_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdatePasswordRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).UpdatePassword(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/UpdatePassword",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).UpdatePassword(ctx, req.(*UpdatePasswordRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ListUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ListUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/ListUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ListUser(ctx, req.(*ListUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_CreateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).CreateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/CreateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).CreateUser(ctx, req.(*CreateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_UpdateUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).UpdateUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/UpdateUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).UpdateUser(ctx, req.(*UpdateUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_RemoveUser_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveUserRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).RemoveUser(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/RemoveUser",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).RemoveUser(ctx, req.(*RemoveUserRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_ListRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).ListRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/ListRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).ListRole(ctx, req.(*ListRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_CreateRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).CreateRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/CreateRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).CreateRole(ctx, req.(*CreateRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_UpdateRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).UpdateRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/UpdateRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).UpdateRole(ctx, req.(*UpdateRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _User_RemoveRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserServer).RemoveRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/user.User/RemoveRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserServer).RemoveRole(ctx, req.(*RemoveRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// User_ServiceDesc is the grpc.ServiceDesc for User service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var User_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "user.User",
	HandlerType: (*UserServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Signup",
			Handler:    _User_Signup_Handler,
		},
		{
			MethodName: "Login",
			Handler:    _User_Login_Handler,
		},
		{
			MethodName: "Logout",
			Handler:    _User_Logout_Handler,
		},
		{
			MethodName: "UpdatePassword",
			Handler:    _User_UpdatePassword_Handler,
		},
		{
			MethodName: "ListUser",
			Handler:    _User_ListUser_Handler,
		},
		{
			MethodName: "CreateUser",
			Handler:    _User_CreateUser_Handler,
		},
		{
			MethodName: "UpdateUser",
			Handler:    _User_UpdateUser_Handler,
		},
		{
			MethodName: "RemoveUser",
			Handler:    _User_RemoveUser_Handler,
		},
		{
			MethodName: "ListRole",
			Handler:    _User_ListRole_Handler,
		},
		{
			MethodName: "CreateRole",
			Handler:    _User_CreateRole_Handler,
		},
		{
			MethodName: "UpdateRole",
			Handler:    _User_UpdateRole_Handler,
		},
		{
			MethodName: "RemoveRole",
			Handler:    _User_RemoveRole_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "user.proto",
}
