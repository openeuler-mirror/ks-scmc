// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package container

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

// ContainerClient is the client API for Container service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ContainerClient interface {
	List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListReply, error)
	Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateReply, error)
	Inspect(ctx context.Context, in *InspectRequest, opts ...grpc.CallOption) (*InspectReply, error)
	Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartReply, error)
	Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopReply, error)
	Remove(ctx context.Context, in *RemoveRequest, opts ...grpc.CallOption) (*RemoveReply, error)
	Restart(ctx context.Context, in *RestartRequest, opts ...grpc.CallOption) (*RestartReply, error)
	Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateReply, error)
	Kill(ctx context.Context, in *KillRequest, opts ...grpc.CallOption) (*KillReply, error)
	Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusReply, error)
	ListTemplate(ctx context.Context, in *ListTemplateRequest, opts ...grpc.CallOption) (*ListTemplateReply, error)
	CreateTemplate(ctx context.Context, in *CreateTemplateRequest, opts ...grpc.CallOption) (*CreateTemplateReply, error)
	UpdateTemplate(ctx context.Context, in *UpdateTemplateRequest, opts ...grpc.CallOption) (*UpdateTemplateReply, error)
	RemoveTemplate(ctx context.Context, in *RemoveTemplateRequest, opts ...grpc.CallOption) (*RemoveTemplateReply, error)
	// 监控历史数据查询
	MonitorHistory(ctx context.Context, in *MonitorHistoryRequest, opts ...grpc.CallOption) (*MonitorHistoryReply, error)
	// 安全配置
	UpdateProcProtect(ctx context.Context, in *UpdateProcProtectRequest, opts ...grpc.CallOption) (*UpdateProcProtectReply, error)
	UpdateNetProcProtect(ctx context.Context, in *UpdateNetProcProtectRequest, opts ...grpc.CallOption) (*UpdateNetProcProtectReply, error)
	UpdateFileProtect(ctx context.Context, in *UpdateFileProtectRequest, opts ...grpc.CallOption) (*UpdateFileProtectReply, error)
	UpdateNetworkRule(ctx context.Context, in *UpdateNetworkRuleRequest, opts ...grpc.CallOption) (*UpdateNetworkRuleReply, error)
}

type containerClient struct {
	cc grpc.ClientConnInterface
}

func NewContainerClient(cc grpc.ClientConnInterface) ContainerClient {
	return &containerClient{cc}
}

func (c *containerClient) List(ctx context.Context, in *ListRequest, opts ...grpc.CallOption) (*ListReply, error) {
	out := new(ListReply)
	err := c.cc.Invoke(ctx, "/container.Container/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Create(ctx context.Context, in *CreateRequest, opts ...grpc.CallOption) (*CreateReply, error) {
	out := new(CreateReply)
	err := c.cc.Invoke(ctx, "/container.Container/Create", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Inspect(ctx context.Context, in *InspectRequest, opts ...grpc.CallOption) (*InspectReply, error) {
	out := new(InspectReply)
	err := c.cc.Invoke(ctx, "/container.Container/Inspect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Start(ctx context.Context, in *StartRequest, opts ...grpc.CallOption) (*StartReply, error) {
	out := new(StartReply)
	err := c.cc.Invoke(ctx, "/container.Container/Start", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Stop(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopReply, error) {
	out := new(StopReply)
	err := c.cc.Invoke(ctx, "/container.Container/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Remove(ctx context.Context, in *RemoveRequest, opts ...grpc.CallOption) (*RemoveReply, error) {
	out := new(RemoveReply)
	err := c.cc.Invoke(ctx, "/container.Container/Remove", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Restart(ctx context.Context, in *RestartRequest, opts ...grpc.CallOption) (*RestartReply, error) {
	out := new(RestartReply)
	err := c.cc.Invoke(ctx, "/container.Container/Restart", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Update(ctx context.Context, in *UpdateRequest, opts ...grpc.CallOption) (*UpdateReply, error) {
	out := new(UpdateReply)
	err := c.cc.Invoke(ctx, "/container.Container/Update", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Kill(ctx context.Context, in *KillRequest, opts ...grpc.CallOption) (*KillReply, error) {
	out := new(KillReply)
	err := c.cc.Invoke(ctx, "/container.Container/Kill", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) Status(ctx context.Context, in *StatusRequest, opts ...grpc.CallOption) (*StatusReply, error) {
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, "/container.Container/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) ListTemplate(ctx context.Context, in *ListTemplateRequest, opts ...grpc.CallOption) (*ListTemplateReply, error) {
	out := new(ListTemplateReply)
	err := c.cc.Invoke(ctx, "/container.Container/ListTemplate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) CreateTemplate(ctx context.Context, in *CreateTemplateRequest, opts ...grpc.CallOption) (*CreateTemplateReply, error) {
	out := new(CreateTemplateReply)
	err := c.cc.Invoke(ctx, "/container.Container/CreateTemplate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) UpdateTemplate(ctx context.Context, in *UpdateTemplateRequest, opts ...grpc.CallOption) (*UpdateTemplateReply, error) {
	out := new(UpdateTemplateReply)
	err := c.cc.Invoke(ctx, "/container.Container/UpdateTemplate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) RemoveTemplate(ctx context.Context, in *RemoveTemplateRequest, opts ...grpc.CallOption) (*RemoveTemplateReply, error) {
	out := new(RemoveTemplateReply)
	err := c.cc.Invoke(ctx, "/container.Container/RemoveTemplate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) MonitorHistory(ctx context.Context, in *MonitorHistoryRequest, opts ...grpc.CallOption) (*MonitorHistoryReply, error) {
	out := new(MonitorHistoryReply)
	err := c.cc.Invoke(ctx, "/container.Container/MonitorHistory", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) UpdateProcProtect(ctx context.Context, in *UpdateProcProtectRequest, opts ...grpc.CallOption) (*UpdateProcProtectReply, error) {
	out := new(UpdateProcProtectReply)
	err := c.cc.Invoke(ctx, "/container.Container/UpdateProcProtect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) UpdateNetProcProtect(ctx context.Context, in *UpdateNetProcProtectRequest, opts ...grpc.CallOption) (*UpdateNetProcProtectReply, error) {
	out := new(UpdateNetProcProtectReply)
	err := c.cc.Invoke(ctx, "/container.Container/UpdateNetProcProtect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) UpdateFileProtect(ctx context.Context, in *UpdateFileProtectRequest, opts ...grpc.CallOption) (*UpdateFileProtectReply, error) {
	out := new(UpdateFileProtectReply)
	err := c.cc.Invoke(ctx, "/container.Container/UpdateFileProtect", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *containerClient) UpdateNetworkRule(ctx context.Context, in *UpdateNetworkRuleRequest, opts ...grpc.CallOption) (*UpdateNetworkRuleReply, error) {
	out := new(UpdateNetworkRuleReply)
	err := c.cc.Invoke(ctx, "/container.Container/UpdateNetworkRule", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ContainerServer is the server API for Container service.
// All implementations must embed UnimplementedContainerServer
// for forward compatibility
type ContainerServer interface {
	List(context.Context, *ListRequest) (*ListReply, error)
	Create(context.Context, *CreateRequest) (*CreateReply, error)
	Inspect(context.Context, *InspectRequest) (*InspectReply, error)
	Start(context.Context, *StartRequest) (*StartReply, error)
	Stop(context.Context, *StopRequest) (*StopReply, error)
	Remove(context.Context, *RemoveRequest) (*RemoveReply, error)
	Restart(context.Context, *RestartRequest) (*RestartReply, error)
	Update(context.Context, *UpdateRequest) (*UpdateReply, error)
	Kill(context.Context, *KillRequest) (*KillReply, error)
	Status(context.Context, *StatusRequest) (*StatusReply, error)
	ListTemplate(context.Context, *ListTemplateRequest) (*ListTemplateReply, error)
	CreateTemplate(context.Context, *CreateTemplateRequest) (*CreateTemplateReply, error)
	UpdateTemplate(context.Context, *UpdateTemplateRequest) (*UpdateTemplateReply, error)
	RemoveTemplate(context.Context, *RemoveTemplateRequest) (*RemoveTemplateReply, error)
	// 监控历史数据查询
	MonitorHistory(context.Context, *MonitorHistoryRequest) (*MonitorHistoryReply, error)
	// 安全配置
	UpdateProcProtect(context.Context, *UpdateProcProtectRequest) (*UpdateProcProtectReply, error)
	UpdateNetProcProtect(context.Context, *UpdateNetProcProtectRequest) (*UpdateNetProcProtectReply, error)
	UpdateFileProtect(context.Context, *UpdateFileProtectRequest) (*UpdateFileProtectReply, error)
	UpdateNetworkRule(context.Context, *UpdateNetworkRuleRequest) (*UpdateNetworkRuleReply, error)
	mustEmbedUnimplementedContainerServer()
}

// UnimplementedContainerServer must be embedded to have forward compatible implementations.
type UnimplementedContainerServer struct {
}

func (UnimplementedContainerServer) List(context.Context, *ListRequest) (*ListReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedContainerServer) Create(context.Context, *CreateRequest) (*CreateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedContainerServer) Inspect(context.Context, *InspectRequest) (*InspectReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Inspect not implemented")
}
func (UnimplementedContainerServer) Start(context.Context, *StartRequest) (*StartReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Start not implemented")
}
func (UnimplementedContainerServer) Stop(context.Context, *StopRequest) (*StopReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedContainerServer) Remove(context.Context, *RemoveRequest) (*RemoveReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Remove not implemented")
}
func (UnimplementedContainerServer) Restart(context.Context, *RestartRequest) (*RestartReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Restart not implemented")
}
func (UnimplementedContainerServer) Update(context.Context, *UpdateRequest) (*UpdateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Update not implemented")
}
func (UnimplementedContainerServer) Kill(context.Context, *KillRequest) (*KillReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Kill not implemented")
}
func (UnimplementedContainerServer) Status(context.Context, *StatusRequest) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedContainerServer) ListTemplate(context.Context, *ListTemplateRequest) (*ListTemplateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListTemplate not implemented")
}
func (UnimplementedContainerServer) CreateTemplate(context.Context, *CreateTemplateRequest) (*CreateTemplateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateTemplate not implemented")
}
func (UnimplementedContainerServer) UpdateTemplate(context.Context, *UpdateTemplateRequest) (*UpdateTemplateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateTemplate not implemented")
}
func (UnimplementedContainerServer) RemoveTemplate(context.Context, *RemoveTemplateRequest) (*RemoveTemplateReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveTemplate not implemented")
}
func (UnimplementedContainerServer) MonitorHistory(context.Context, *MonitorHistoryRequest) (*MonitorHistoryReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MonitorHistory not implemented")
}
func (UnimplementedContainerServer) UpdateProcProtect(context.Context, *UpdateProcProtectRequest) (*UpdateProcProtectReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateProcProtect not implemented")
}
func (UnimplementedContainerServer) UpdateNetProcProtect(context.Context, *UpdateNetProcProtectRequest) (*UpdateNetProcProtectReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateNetProcProtect not implemented")
}
func (UnimplementedContainerServer) UpdateFileProtect(context.Context, *UpdateFileProtectRequest) (*UpdateFileProtectReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateFileProtect not implemented")
}
func (UnimplementedContainerServer) UpdateNetworkRule(context.Context, *UpdateNetworkRuleRequest) (*UpdateNetworkRuleReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateNetworkRule not implemented")
}
func (UnimplementedContainerServer) mustEmbedUnimplementedContainerServer() {}

// UnsafeContainerServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ContainerServer will
// result in compilation errors.
type UnsafeContainerServer interface {
	mustEmbedUnimplementedContainerServer()
}

func RegisterContainerServer(s grpc.ServiceRegistrar, srv ContainerServer) {
	s.RegisterService(&Container_ServiceDesc, srv)
}

func _Container_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).List(ctx, req.(*ListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Create",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Create(ctx, req.(*CreateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Inspect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InspectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Inspect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Inspect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Inspect(ctx, req.(*InspectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Start_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Start(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Start",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Start(ctx, req.(*StartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Stop(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Remove_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Remove(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Remove",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Remove(ctx, req.(*RemoveRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Restart_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RestartRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Restart(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Restart",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Restart(ctx, req.(*RestartRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Update_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Update(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Update",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Update(ctx, req.(*UpdateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Kill_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(KillRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Kill(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Kill",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Kill(ctx, req.(*KillRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).Status(ctx, req.(*StatusRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_ListTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).ListTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/ListTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).ListTemplate(ctx, req.(*ListTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_CreateTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).CreateTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/CreateTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).CreateTemplate(ctx, req.(*CreateTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_UpdateTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).UpdateTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/UpdateTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).UpdateTemplate(ctx, req.(*UpdateTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_RemoveTemplate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RemoveTemplateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).RemoveTemplate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/RemoveTemplate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).RemoveTemplate(ctx, req.(*RemoveTemplateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_MonitorHistory_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MonitorHistoryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).MonitorHistory(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/MonitorHistory",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).MonitorHistory(ctx, req.(*MonitorHistoryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_UpdateProcProtect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateProcProtectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).UpdateProcProtect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/UpdateProcProtect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).UpdateProcProtect(ctx, req.(*UpdateProcProtectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_UpdateNetProcProtect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateNetProcProtectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).UpdateNetProcProtect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/UpdateNetProcProtect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).UpdateNetProcProtect(ctx, req.(*UpdateNetProcProtectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_UpdateFileProtect_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateFileProtectRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).UpdateFileProtect(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/UpdateFileProtect",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).UpdateFileProtect(ctx, req.(*UpdateFileProtectRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Container_UpdateNetworkRule_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateNetworkRuleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ContainerServer).UpdateNetworkRule(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/container.Container/UpdateNetworkRule",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ContainerServer).UpdateNetworkRule(ctx, req.(*UpdateNetworkRuleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Container_ServiceDesc is the grpc.ServiceDesc for Container service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Container_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "container.Container",
	HandlerType: (*ContainerServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "List",
			Handler:    _Container_List_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _Container_Create_Handler,
		},
		{
			MethodName: "Inspect",
			Handler:    _Container_Inspect_Handler,
		},
		{
			MethodName: "Start",
			Handler:    _Container_Start_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _Container_Stop_Handler,
		},
		{
			MethodName: "Remove",
			Handler:    _Container_Remove_Handler,
		},
		{
			MethodName: "Restart",
			Handler:    _Container_Restart_Handler,
		},
		{
			MethodName: "Update",
			Handler:    _Container_Update_Handler,
		},
		{
			MethodName: "Kill",
			Handler:    _Container_Kill_Handler,
		},
		{
			MethodName: "Status",
			Handler:    _Container_Status_Handler,
		},
		{
			MethodName: "ListTemplate",
			Handler:    _Container_ListTemplate_Handler,
		},
		{
			MethodName: "CreateTemplate",
			Handler:    _Container_CreateTemplate_Handler,
		},
		{
			MethodName: "UpdateTemplate",
			Handler:    _Container_UpdateTemplate_Handler,
		},
		{
			MethodName: "RemoveTemplate",
			Handler:    _Container_RemoveTemplate_Handler,
		},
		{
			MethodName: "MonitorHistory",
			Handler:    _Container_MonitorHistory_Handler,
		},
		{
			MethodName: "UpdateProcProtect",
			Handler:    _Container_UpdateProcProtect_Handler,
		},
		{
			MethodName: "UpdateNetProcProtect",
			Handler:    _Container_UpdateNetProcProtect_Handler,
		},
		{
			MethodName: "UpdateFileProtect",
			Handler:    _Container_UpdateFileProtect_Handler,
		},
		{
			MethodName: "UpdateNetworkRule",
			Handler:    _Container_UpdateNetworkRule_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "container.proto",
}
