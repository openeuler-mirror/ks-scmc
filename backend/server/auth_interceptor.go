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
	// PERMISSION_SYS_INFO_READ  PERMISSION = 1  // 系统-概要-查看
	// PERMISSION_SYS_INFO_WRITE PERMISSION = 2  // 系统-概要-管理(保留设计)
	// PERMISSION_SYS_PERM_READ  PERMISSION = 11 // 系统-权限-查看
	// PERMISSION_SYS_PERM_WRITE PERMISSION = 12 // 系统-权限-管理
	case "/user.User/ListRole":
		return pb.PERMISSION_SYS_PERM_READ
	case "/user.User/CreateRole", "/user.User/UpdateRole", "/user.User/RemoveUser":
		return pb.PERMISSION_SYS_PERM_WRITE
	// 容器
	// PERMISSION_CONTAINER_INFO_READ  PERMISSION = 1001 // 容器-信息-查看
	// PERMISSION_CONTAINER_INFO_WRITE PERMISSION = 1002 // 容器-信息-管理
	// PERMISSION_CONTAINER_TEMP_READ  PERMISSION = 1031 // 容器-模板-查看
	// PERMISSION_CONTAINER_TEMP_WRITE PERMISSION = 1032 // 容器-模板-管理
	case "/container.Container/List", "/container.Container/Inspect", "/container.Container/Status":
		return pb.PERMISSION_CONTAINER_INFO_READ
	case "/container.Container/Create", "/container.Container/Start", "/container.Container/Stop",
		"/container.Container/Remove", "/container.Container/Restart", "/container.Container/Update",
		"/container.Container/Kill":
		return pb.PERMISSION_CONTAINER_INFO_WRITE
	case "/container.Container/ListTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_READ
	case "/container.Container/CreateTemplate", "/container.Container/UpdateTemplate", "/container.Container/RemoveTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_WRITE
	// 节点
	// PERMISSION_NODE_INFO_READ  PERMISSION = 2001 // 节点-信息-查看
	// PERMISSION_NODE_INFO_WRITE PERMISSION = 2002 // 节点-信息-管理
	case "/node.Node/List", "/node.Node/Status":
		return pb.PERMISSION_NODE_INFO_READ
	case "/node.Node/Create", "/node.Node/Update", "/node.Node/Remove":
		return pb.PERMISSION_NODE_INFO_WRITE
	// 镜像
	// PERMISSION_IMAGE_INFO_READ  PERMISSION = 3001 // 镜像-信息-查看
	// PERMISSION_IMAGE_INFO_WRITE PERMISSION = 3002 // 镜像-信息-管理
	case "/image.Image/List", "/image.Image/ListDB":
		return pb.PERMISSION_IMAGE_INFO_READ
	case "/image.Image/Remove", "/image.Image/Approve", "/image.Image/Update", "/image.Image/Upload", "/image.Image/Download":
		return pb.PERMISSION_IMAGE_INFO_WRITE
	// 审计
	// PERMISSION_AUDIT_APPROVE_READ  PERMISSION = 4001 // 审计-审核-查看
	// PERMISSION_AUDIT_APPROVE_WRITE PERMISSION = 4002 // 审计-审核-查看
	// PERMISSION_AUTID_WARN_READ     PERMISSION = 4011 // 审计-告警-查看
	// PERMISSION_AUTID_LOG_READ      PERMISSION = 4021 // 审计-日志-查看
	// PERMISSION_NONE PERMISSION = 0
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
