package server

import (
	"fmt"
	"scmc/model"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	"scmc/rpc/pb/node"
	"scmc/rpc/pb/user"
)

type RuntimeLogWritter struct{}

func (RuntimeLogWritter) Login(r *user.LoginRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_USER_LOGIN),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("用户=%v", r.Username),
	}
}

func (RuntimeLogWritter) Logout(r *user.LogoutRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_USER_LOGOUT),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}
}

func (RuntimeLogWritter) UpdatePassword(r *user.UpdatePasswordRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_PASSWORD),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}
}

func (RuntimeLogWritter) CreateRole(r *user.CreateRoleRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.RoleInfo != nil {
		l.Target = fmt.Sprintf("角色=%s", r.RoleInfo.Name)
		l.Detail = fmt.Sprintf("角色=%s 可编辑=%v 权限数=%v", r.RoleInfo.Name, r.RoleInfo.IsEditable, len(r.RoleInfo.Perms))
	}

	return l
}

func (RuntimeLogWritter) RemoveRole(r *user.RemoveRoleRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("角色ID=%v", r.RoleId),
	}
}

func (RuntimeLogWritter) UpdateRole(r *user.UpdateRoleRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.RoleInfo != nil {
		l.Target = fmt.Sprintf("角色ID=%v", r.RoleInfo.Id)
		l.Detail = fmt.Sprintf("权限数=%v", len(r.RoleInfo.Perms))
	}

	return l
}

func (RuntimeLogWritter) CreateUser(r *user.CreateUserRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_ROLE),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.UserInfo != nil {
		l.Target = fmt.Sprintf("用户=%s", r.UserInfo.LoginName)
		l.Detail = fmt.Sprintf("用户=%s 启用=%v 可编辑=%v", r.UserInfo.LoginName, r.UserInfo.IsActive, r.UserInfo.IsEditable)
	}

	return l
}

func (RuntimeLogWritter) RemoveUser(r *user.RemoveUserRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_USER),
		EventModule: int64(logging.EVENT_MODULE_USER),
		Target:      fmt.Sprintf("用户数=%v", len(r.UserIds)),
		Detail:      fmt.Sprintf("用户ID=%v", r.UserIds),
	}
}

func (RuntimeLogWritter) UpdateUser(r *user.UpdateUserRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_USER),
		EventModule: int64(logging.EVENT_MODULE_USER),
	}

	if r.UserInfo != nil {
		l.Target = fmt.Sprintf("用户ID=%v", r.UserInfo.Id)
		l.Detail = fmt.Sprintf("启用=%v 角色ID=%v", r.UserInfo.IsActive, r.UserInfo.RoleId)
	}

	return l
}

func (RuntimeLogWritter) CreateNode(r *node.CreateRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点=%s", r.Name),
		Detail:      fmt.Sprintf("节点=%s 地址=%s 备注=%s", r.Name, r.Address, r.Comment),
	}
}

func (RuntimeLogWritter) RemoveNode(r *node.RemoveRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("节点ID=%v", r.Ids),
	}
}

func (RuntimeLogWritter) UpdateNode(r *node.UpdateRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPDATE_NODE),
		EventModule: int64(logging.EVENT_MODULE_NODE),
		Target:      fmt.Sprintf("节点ID=%v", r.NodeId),
		Detail:      fmt.Sprintf("节点=%s 备注=%s", r.Name, r.Comment),
	}
}

func (RuntimeLogWritter) CreateContainer(r *container.CreateRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_CREATE_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
	}

	if r.Configs != nil {
		l.Target = fmt.Sprintf("容器名=%v", r.Configs.Name)
	}

	return l
}

func (RuntimeLogWritter) RemoveContainer(r *container.RemoveRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (RuntimeLogWritter) StartContainer(r *container.StartRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_START_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (RuntimeLogWritter) StopContainer(r *container.StopRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_STOP_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (RuntimeLogWritter) RestartContainer(r *container.RestartRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_RESTART_CONTAINER),
		EventModule: int64(logging.EVENT_MODULE_CONTAINER),
		Target:      fmt.Sprintf("容器数=%v", len(r.Ids)),
		Detail:      fmt.Sprintf("容器ID=%v", r.Ids),
	}
}

func (RuntimeLogWritter) UploadImage(r *image.UploadRequest) *model.RuntimeLog {
	l := &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_UPLOAD_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
	}

	if r.Info != nil {
		l.Target = fmt.Sprintf("镜像版本=%s:%s", r.Info.Name, r.Info.Version)
	}

	return l
}

func (RuntimeLogWritter) RemoveImage(r *image.RemoveRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_REMOVE_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Target:      fmt.Sprintf("镜像数=%v", len(r.ImageIds)),
		Detail:      fmt.Sprintf("镜像ID=%v", r.ImageIds),
	}
}

func (RuntimeLogWritter) ApproveImage(r *image.ApproveRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_APPROVE_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Detail:      fmt.Sprintf("镜像ID=%v 通过=%v", r.ImageId, r.Approve),
	}
}

func (RuntimeLogWritter) DownloadImage(r *image.DownloadRequest) *model.RuntimeLog {
	return &model.RuntimeLog{
		EventType:   int64(logging.EVENT_TYPE_DOWNLOAD_IMAGE),
		EventModule: int64(logging.EVENT_MODULE_IMAGE),
		Detail:      fmt.Sprintf("镜像ID=%v", r.ImageId),
	}
}
