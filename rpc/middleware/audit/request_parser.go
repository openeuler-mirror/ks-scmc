package audit

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"scmc/model/db"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	"scmc/rpc/pb/node"
	"scmc/rpc/pb/user"
)

type parser struct{}

func (p parser) Parse(v interface{}) *db.RuntimeLog {
	switch m := v.(type) {
	case *user.LoginRequest:
		return p.Login(m)
	case *user.LogoutRequest:
		return p.Logout(m)
	case *user.UpdatePasswordRequest:
		return p.UpdatePassword(m)
	case *user.CreateUserRequest:
		return p.CreateUser(m)
	case *user.RemoveUserRequest:
		return p.RemoveUser(m)
	case *user.UpdateUserRequest:
		return p.UpdateUser(m)
	case *user.CreateRoleRequest:
		return p.CreateRole(m)
	case *user.RemoveRoleRequest:
		return p.RemoveRole(m)
	case *user.UpdateRoleRequest:
		return p.UpdateRole(m)
	case *node.CreateRequest:
		return p.CreateNode(m)
	case *node.UpdateRequest:
		return p.UpdateNode(m)
	case *node.RemoveRequest:
		return p.RemoveNode(m)
	case *container.RemoveRequest:
		return p.RemoveContainer(m)
	case *container.RestartRequest:
		return p.RestartContainer(m)
	case *container.CreateRequest:
		return p.CreateContainer(m)
	case *container.StartRequest:
		return p.StartContainer(m)
	case *container.StopRequest:
		return p.StopContainer(m)
	case *image.RemoveRequest:
		return p.RemoveImage(m)
	case *image.ApproveRequest:
		return p.ApproveImage(m)
	case *image.UploadRequest:
		return p.UploadImage(m)
	case *image.UpdateRequest:
		return p.UpdateImage(m)
	case *image.DownloadRequest:
		return p.DownloadImage(m)
	case nil:
		return nil
	default:
		log.Debugf("ignore message type=%T", m)
		return nil
	}
}

func (parser) Login(r *user.LoginRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_USER_LOGIN),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("用户=%v", r.Username),
	}
}

func (parser) Logout(r *user.LogoutRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_USER_LOGOUT),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}
}

func (parser) UpdatePassword(r *user.UpdatePasswordRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_PASSWORD),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}
}

func (parser) CreateRole(r *user.CreateRoleRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.RoleInfo != nil {
		l.Target = fmt.Sprintf("角色=%s", r.RoleInfo.Name)
		l.Detail = fmt.Sprintf("角色=%s 可编辑=%v 权限数=%v", r.RoleInfo.Name, r.RoleInfo.IsEditable, len(r.RoleInfo.Perms))
	}

	return l
}

func (parser) RemoveRole(r *user.RemoveRoleRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("角色ID=%v", r.RoleId),
	}
}

func (parser) UpdateRole(r *user.UpdateRoleRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.RoleInfo != nil {
		l.Target = fmt.Sprintf("角色ID=%v", r.RoleInfo.Id)
		l.Detail = fmt.Sprintf("权限数=%v", len(r.RoleInfo.Perms))
	}

	return l
}

func (parser) CreateUser(r *user.CreateUserRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.UserInfo != nil {
		l.Target = fmt.Sprintf("用户=%s", r.UserInfo.LoginName)
		l.Detail = fmt.Sprintf("用户=%s 启用=%v 可编辑=%v", r.UserInfo.LoginName, r.UserInfo.IsActive, r.UserInfo.IsEditable)
	}

	return l
}

func (parser) RemoveUser(r *user.RemoveUserRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_USER),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("用户数=%v", len(r.UserIds)),
		Detail:      fmt.Sprintf("用户ID=%v", r.UserIds),
	}
}

func (parser) UpdateUser(r *user.UpdateUserRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_USER),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.UserInfo != nil {
		l.Target = fmt.Sprintf("用户ID=%v", r.UserInfo.Id)
		l.Detail = fmt.Sprintf("启用=%v 角色ID=%v", r.UserInfo.IsActive, r.UserInfo.RoleId)
	}

	return l
}

func (parser) CreateNode(r *node.CreateRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点=%s", r.Name),
		Detail:      fmt.Sprintf("节点=%s 地址=%s 备注=%s", r.Name, r.Address, r.Comment),
	}
}

func (parser) RemoveNode(r *node.RemoveRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("节点ID=%v", r.Ids),
	}
}

func (parser) UpdateNode(r *node.UpdateRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点ID=%v", r.NodeId),
		Detail:      fmt.Sprintf("节点=%s 备注=%s", r.Name, r.Comment),
	}
}

func (parser) CreateContainer(r *container.CreateRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
	}

	if r.Configs != nil {
		l.Target = fmt.Sprintf("容器名=%v", r.Configs.Name)
	}

	return l
}

func (parser) RemoveContainer(r *container.RemoveRequest) *db.RuntimeLog {
	containerNum := len(r.Ids)
	if containerNum > 0 {
		containerNum = len(r.Ids[0].ContainerIds)
	}

	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", containerNum),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (parser) StartContainer(r *container.StartRequest) *db.RuntimeLog {
	containerNum := len(r.Ids)
	if containerNum > 0 {
		containerNum = len(r.Ids[0].ContainerIds)
	}

	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_START_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", containerNum),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (parser) StopContainer(r *container.StopRequest) *db.RuntimeLog {
	containerNum := len(r.Ids)
	if containerNum > 0 {
		containerNum = len(r.Ids[0].ContainerIds)
	}

	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_STOP_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", containerNum),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (parser) RestartContainer(r *container.RestartRequest) *db.RuntimeLog {
	containerNum := len(r.Ids)
	if containerNum > 0 {
		containerNum = len(r.Ids[0].ContainerIds)
	}

	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_RESTART_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", containerNum),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (parser) UploadImage(r *image.UploadRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPLOAD_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
	}

	if r.Info != nil {
		l.Target = fmt.Sprintf("镜像版本=%s:%s", r.Info.Name, r.Info.Version)
	}

	return l
}

func (parser) UpdateImage(r *image.UpdateRequest) *db.RuntimeLog {
	l := &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
	}

	if r.Info != nil {
		l.Target = fmt.Sprintf("镜像版本=%s:%s", r.Info.Name, r.Info.Version)
	}

	return l
}

func (parser) RemoveImage(r *image.RemoveRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Target:      fmt.Sprintf("镜像数=%v", len(r.ImageIds)),
		Detail:      fmt.Sprintf("镜像ID=%v", r.ImageIds),
	}
}

func (parser) ApproveImage(r *image.ApproveRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_APPROVE_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Detail:      fmt.Sprintf("镜像ID=%v 通过=%v", r.ImageId, r.Approve),
	}
}

func (parser) DownloadImage(r *image.DownloadRequest) *db.RuntimeLog {
	return &db.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_DOWNLOAD_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Detail:      fmt.Sprintf("镜像ID=%v", r.ImageId),
	}
}

type WrapperStream struct {
	Ss  grpc.ServerStream
	Ctx context.Context
	msg int
}

func NewWrapperStream(ss grpc.ServerStream, ctx context.Context) grpc.ServerStream {
	return &WrapperStream{
		Ss:  ss,
		Ctx: ctx,
		msg: 0,
	}
}

func (w *WrapperStream) Context() context.Context {
	return w.Ctx
}

func (w *WrapperStream) RecvMsg(msg interface{}) error {
	err := w.Ss.RecvMsg(msg)
	if w.msg == 0 {
		ctx := context.WithValue(w.Ctx, "message", msg)
		w.Ctx = ctx
		w.msg++
	}

	return err
}

func (w *WrapperStream) SendMsg(msg interface{}) error {
	err := w.Ss.SendMsg(msg)
	if w.msg == 0 {
		ctx := context.WithValue(w.Ctx, "message", msg)
		w.Ctx = ctx
		w.msg++
	}
	return err
}

func (w *WrapperStream) SendHeader(md metadata.MD) error {
	return w.Ss.SendHeader(md)
}

func (w *WrapperStream) SetHeader(md metadata.MD) error {
	return w.Ss.SetHeader(md)
}

func (w *WrapperStream) SetTrailer(md metadata.MD) {
	w.Ss.SetTrailer(md)
}
