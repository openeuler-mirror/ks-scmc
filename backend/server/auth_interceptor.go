package server

import (
	"context"
	"encoding/json"
	"strings"

	"scmc/common"
	"scmc/model"
	"scmc/rpc"

	pb "scmc/rpc/pb/user"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
func (auth *AuthInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		if info.FullMethod == "/user.User/Login" || info.FullMethod == "/user.User/Signup" {
			reply, err = handler(ctx, req)
			return reply, err
		} else {
			userID, accessToken, err := getAuthInfo(ctx)
			if err != nil {
				return nil, err
			}

			if common.Config.Controller.CheckAuth {
				if err := auth.checkAuthentication(userID, accessToken); err != nil {
					// log.Infof("AuthInterceptor error: %v", e)
					return nil, err
				}
			}

			if common.Config.Controller.CheckPerm {
				if err = auth.checkAuthorization(userID, info.FullMethod); err != nil {
					// log.Infof("AuthInterceptor error: %v", err)
					return nil, err
				}
			}
		}

		reply, err = handler(ctx, req)
		return reply, err
	}
}

func (auth *AuthInterceptor) Streams() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if info.FullMethod == "/user.User/Login" || info.FullMethod == "/user.User/Signup" {
			err := handler(srv, ss)
			return err
		} else {
			userID, accessToken, err := getAuthInfo(ss.Context())
			if err != nil {
				return err
			}

			if common.Config.Controller.CheckAuth {
				if err := auth.checkAuthentication(userID, accessToken); err != nil {
					// log.Infof("AuthInterceptor error: %v", e)
					return err
				}
			}

			if common.Config.Controller.CheckPerm {
				if err = auth.checkAuthorization(userID, info.FullMethod); err != nil {
					// log.Infof("AuthInterceptor error: %v", err)
					return err
				}
			}
		}

		err := handler(srv, ss)
		return err
	}
}

func (auth *AuthInterceptor) checkAuthentication(userID, accessToken string) error {

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

func (auth *AuthInterceptor) checkAuthorization(userID, method string) error {
	log.Debugf("check user %s method %s permission", userID, method)
	ok, err := CheckUserPermission(userID, method)
	if err != nil {
		log.Info("model.CheckUserPermission error")
		return rpc.ErrInternal
	} else if !ok {
		log.Info("role permission denied ")
		return rpc.ErrPermissionDenied
	}

	return nil
}

func getAuthInfo(ctx context.Context) (userID, accessToken string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Infof("get metadata from incoming context failed")
		return userID, accessToken, rpc.ErrUnauthenticated
	}

	values := md["authorization"]
	if len(values) == 0 {
		log.Infof("'authorization' not exist in request metadata.")
		return userID, accessToken, rpc.ErrUnauthenticated
	}

	authorization := values[0]
	i := strings.IndexRune(authorization, AuthKeySeprator)
	if i == -1 {
		log.Infof("invalid authorization metadata: %v", authorization)
		return userID, accessToken, rpc.ErrUnauthenticated
	}

	userID, accessToken = authorization[:i], authorization[i+1:]
	return userID, accessToken, nil
}

func getPermID(method string) pb.PERMISSION {
	switch method {
	// 系统
	// SYS_INFO_READ  = 1;   // 系统-概要-查看
	// SYS_INFO_WRITE = 2;   // 系统-概要-管理(保留设计)
	// SYS_PERM_READ  = 11;  // 系统-权限-查看
	// SYS_PERM_WRITE = 12;  // 系统-权限-管理
	case "/user.User/ListRole",
		"/user.User/ListUser":
		return pb.PERMISSION_SYS_PERM_READ
	case "/user.User/CreateRole", "/user.User/UpdateRole", "/user.User/RemoveRole",
		"/user.User/CreateUser", "/user.User/UpdateUser", "/user.User/RemoveUser":
		return pb.PERMISSION_SYS_PERM_WRITE
	// 容器
	// CONTAINER_INFO_READ  = 1001;  // 容器-信息-查看
	// CONTAINER_INFO_WRITE = 1002;  // 容器-信息-管理
	// CONTAINER_TEMP_READ  = 1031;  // 容器-模板-查看
	// CONTAINER_TEMP_WRITE = 1032;  // 容器-模板-管理
	case "/container.Container/List", "/container.Container/Inspect", "/container.Container/Status",
		// 容器备份
		"/container.Container/ListBackup", "/container.Container/GetBackupJob",
		"/container.Container/MonitorHistory",
		// 容器网络
		"/network.Network/List", "/network.Network/ListIPtables",
		// 容器安全
		"/security.Security/ListProcProtection", "/security.Security/ListFileProtection":
		return pb.PERMISSION_CONTAINER_INFO_READ
	case "/container.Container/Create", "/container.Container/Start", "/container.Container/Stop",
		"/container.Container/Remove", "/container.Container/Restart", "/container.Container/Update",
		"/container.Container/Kill",
		// 容器备份
		"/container.Container/CreateBackup", "/container.Container/UpdateBackup", "/container.Container/ResumeBackup",
		"/container.Container/RemoveBackup",
		"/container.Container/AddBackupJob", "/container.Container/DelBackupJob",
		// 容器网络
		"/network.Network/Connect", "/network.Network/Disconnect", "/network.Network/EnableIPtables",
		"/network.Network/CreateIPtables", "/network.Network/ModifyIPtables", "/network.Network/RemoveIPtables",
		"/network.Network/Create", "/network.Network/Remove",
		// 容器安全
		"/security.Security/UpdateProcProtection", "/security.Security/UpdateFileProtection":
		return pb.PERMISSION_CONTAINER_INFO_WRITE
	case "/container.Container/ListTemplate", "/container.Container/InspectTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_READ
	case "/container.Container/CreateTemplate", "/container.Container/UpdateTemplate", "/container.Container/RemoveTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_WRITE
	// 节点
	// NODE_INFO_READ  = 2001;  // 节点-信息-查看
	// NODE_INFO_WRITE = 2002;  // 节点-信息-管理
	case "/node.Node/List", "/node.Node/Status":
		return pb.PERMISSION_NODE_INFO_READ
	case "/node.Node/Create", "/node.Node/Update", "/node.Node/Remove",
		"/node.Node/UpdateFileProtect", "/node.Node/UpdateNetworkRule":
		return pb.PERMISSION_NODE_INFO_WRITE
	// 镜像
	// IMAGE_INFO_READ  = 3001;  // 镜像-信息-查看
	// IMAGE_INFO_WRITE = 3002;  // 镜像-信息-管理
	case "/image.Image/List", "/image.Image/ListDB":
		return pb.PERMISSION_IMAGE_INFO_READ
	case "/image.Image/Remove", "/image.Image/Update", "/image.Image/Upload", "/image.Image/Download":
		return pb.PERMISSION_IMAGE_INFO_WRITE
	// 审计
	// AUDIT_APPROVE_READ  = 4001;  // 审计-审核-查看
	// AUDIT_APPROVE_WRITE = 4002;  // 审计-审核-管理
	// AUTID_WARN_READ     = 4011;  // 审计-告警-查看
	// AUTID_LOG_READ      = 4021;  // 审计-日志-查看
	case "/image.Image/Approve":
		return pb.PERMISSION_AUDIT_APPROVE_WRITE
	case "/logging.Logging/ListRuntime":
		return pb.PERMISSION_AUTID_WARN_READ
	case "/logging.Logging/ListWarn", "/logging.Logging/ReadWarn":
		return pb.PERMISSION_AUTID_LOG_READ
	// NONE = 0;
	default:
		return pb.PERMISSION_NONE
	}

}

func CheckUserPermission(userID string, method string) (bool, error) {
	permID := int32(getPermID(method))
	log.Debugf("permission name is %s", permID)
	if permID == 0 {
		log.Info("cannot recognize perm for method=%s", method)
		return true, nil
	}
	userRole, err := model.QueryRoleByUserID(userID)
	if err != nil {
		return false, err
	}

	var perms []pb.Permission
	err = json.Unmarshal([]byte(userRole.PermsJson), &perms)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		if p.Id == permID && p.Allow {
			return true, nil
		}

	}
	return false, nil
}
