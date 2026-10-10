package auth

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"scmc/common"
	"scmc/model/db"
	"scmc/model/session"
	"scmc/rpc"
	"scmc/rpc/middleware"
	pb "scmc/rpc/pb/user"
)

type ContextKey string

const (
	keyPermission  ContextKey = "PERMS"
	keyUserInfo    ContextKey = "USER"
	keyUserID      ContextKey = "USER_ID"
	keyUserSession ContextKey = "USER_SESSION"
)

func UserInfoFromContext(ctx context.Context) (*db.UserRoleInfo, bool) {
	v, ok := ctx.Value(keyUserInfo).(*db.UserRoleInfo)
	return v, ok
}

// UserSessionFromContext returns user_id, user_session from context
func UserSessionFromContext(ctx context.Context) (string, string, bool) {
	u, ok0 := ctx.Value(keyUserID).(string)
	s, ok1 := ctx.Value(keyUserSession).(string)
	return u, s, ok0 && ok1
}

func UserPermFromContext(ctx context.Context) ([]*pb.Permission, bool) {
	v, ok := ctx.Value(keyPermission).([]*pb.Permission)
	return v, ok
}

func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		if !common.NeedCheckAuth() || !rpcNeedCheckAuth(info.FullMethod) {
			goto handle
		} else {
			userID, sessionKey, err := getAuthInfo(ctx)
			if err != nil {
				return nil, err
			}

			if err := checkUserSession(userID, sessionKey); err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, keyUserID, userID)
			ctx = context.WithValue(ctx, keyUserSession, sessionKey)

			if !common.NeedCheckPerm() || !rpcNeedCheckPerm(info.FullMethod) {
				goto handle
			}

			ctx, err = checkUserPermission(ctx, userID, info.FullMethod)
			if err != nil {
				log.Infof("AuthInterceptor method=%v user=%v error=%v", info.FullMethod, userID, err)
				return nil, err
			}
		}
	handle:
		return handler(ctx, req)
	}
}

func StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !common.NeedCheckAuth() || !rpcNeedCheckAuth(info.FullMethod) {
			return handler(srv, stream)
		} else {
			ctx := stream.Context()
			userID, sessionKey, err := getAuthInfo(ctx)
			if err != nil {
				return err
			}

			if err := checkUserSession(userID, sessionKey); err != nil {
				return err
			}
			ctx = context.WithValue(ctx, keyUserID, userID)
			ctx = context.WithValue(ctx, keyUserSession, sessionKey)

			if !common.NeedCheckPerm() || !rpcNeedCheckPerm(info.FullMethod) {
				return handler(srv, middleware.NewServerStreamWrapper(stream, ctx))
			}

			ctx, err = checkUserPermission(ctx, userID, info.FullMethod)
			if err != nil {
				log.Infof("AuthInterceptor method=%v user=%v error=%v", info.FullMethod, userID, err)
				return err
			}
			return handler(srv, middleware.NewServerStreamWrapper(stream, ctx))
		}
	}
}

func rpcNeedCheckAuth(method string) bool {
	switch method {
	case "/user.User/Login", "/user.User/Signup", "/sys.System/Ping":
		return false
	}
	return true
}

func rpcNeedCheckPerm(method string) bool {
	switch method {
	case "/user.User/Logout",
		"/user.User/UpdatePassword",
		"/sys.System/Subscribe":
		return false
	}
	return true
}

func checkUserSession(userID, sessionKey string) error {
	now := time.Now()
	s, err := db.CheckUserSession(userID, sessionKey)
	if err != nil {
		log.Info("db.CheckUserSession error")
		return rpc.ErrInternal
	} else if s == nil || s.ExpiredAt <= now.Unix() {
		log.Info("invalid user session")
		return rpc.ErrUnauthenticated
	}

	session.Timers.Reset(sessionKey, s.ExpiredAt)
	return nil
}

func checkUserPermission(ctx context.Context, userID, method string) (context.Context, error) {
	permID := getPermID(method)
	if permID == 0 {
		log.Warnf("cannot recognize perm for method=%s", method)
		return nil, rpc.ErrPermissionDenied
	}

	userRole, err := db.QueryUserRole(userID)
	if err != nil {
		log.Warnf("query info of user=%v err=%v", userID, err)
		return nil, rpc.ErrPermissionDenied
	}

	var perms []*pb.Permission
	err = json.Unmarshal([]byte(userRole.PermsJSON), &perms)
	if err != nil {
		log.Warnf("unmarshal permission json err=%v", err)
		return nil, rpc.ErrPermissionDenied
	}

	if !hasPerm(permID, perms) {
		return nil, rpc.ErrPermissionDenied
	}

	c := context.WithValue(ctx, keyUserInfo, userRole)
	c = context.WithValue(c, keyPermission, perms)
	return c, nil
}

func getAuthInfo(ctx context.Context) (userID, sessionKey string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Infof("get metadata from incoming context failed")
		return userID, sessionKey, rpc.ErrUnauthenticated
	}

	values := md["authorization"]
	if len(values) == 0 {
		log.Infof("'authorization' not exist in request metadata.")
		return userID, sessionKey, rpc.ErrUnauthenticated
	}

	authorization := values[0]
	i := strings.IndexRune(authorization, common.UserSessionySeprator)
	if i == -1 {
		log.Infof("invalid authorization metadata: %v", authorization)
		return userID, sessionKey, rpc.ErrUnauthenticated
	}

	userID, sessionKey = authorization[:i], authorization[i+1:]
	return userID, sessionKey, nil
}

func getPermID(method string) pb.PERMISSION {
	switch method {
	case "/user.User/ListRole",
		"/user.User/ListUser":
		return pb.PERMISSION_SYS_PERM_READ
	case "/user.User/CreateRole",
		"/user.User/UpdateRole",
		"/user.User/RemoveRole",
		"/user.User/CreateUser",
		"/user.User/UpdateUser",
		"/user.User/RemoveUser":
		return pb.PERMISSION_SYS_PERM_WRITE
	case "/container.Container/List",
		"/container.Container/Inspect",
		"/container.Container/Status",
		"/container.Container/ListBackup",
		"/container.Container/GetBackupJob",
		"/container.Container/MonitorHistory",
		"/network.Network/List",
		"/network.Network/ListIPtables",
		"/security.Security/ListProcProtection",
		"/security.Security/ListFileProtection":
		return pb.PERMISSION_CONTAINER_INFO_READ
	case "/container.Container/Create",
		"/container.Container/Start",
		"/container.Container/Stop",
		"/container.Container/Remove",
		"/container.Container/Restart",
		"/container.Container/Kill",
		"/container.Container/CreateBackup",
		"/container.Container/UpdateBackup",
		"/container.Container/ResumeBackup",
		"/container.Container/RemoveBackup",
		"/container.Container/AddBackupJob",
		"/container.Container/DelBackupJob",
		"/network.Network/Connect",
		"/network.Network/Disconnect",
		"/network.Network/EnableIPtables",
		"/network.Network/CreateIPtables",
		"/network.Network/ModifyIPtables",
		"/network.Network/RemoveIPtables",
		"/network.Network/Create",
		"/network.Network/Remove",
		"/security.Security/UpdateProcProtection",
		"/security.Security/UpdateFileProtection":
		return pb.PERMISSION_CONTAINER_INFO_WRITE
	case "/container.Container/Update":
		return pb.PERMISSION_CONTAINER_CONF_WRITE
	case "/container.Container/ListTemplate",
		"/container.Container/InspectTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_READ
	case "/container.Container/CreateTemplate",
		"/container.Container/UpdateTemplate",
		"/container.Container/RemoveTemplate":
		return pb.PERMISSION_CONTAINER_TEMP_WRITE
	case "/node.Node/List",
		"/node.Node/Status":
		return pb.PERMISSION_NODE_INFO_READ
	case "/node.Node/Create",
		"/node.Node/Update",
		"/node.Node/Remove",
		"/node.Node/UpdateFileProtect",
		"/node.Node/UpdateNetworkRule":
		return pb.PERMISSION_NODE_INFO_WRITE
	case "/image.Image/List",
		"/image.Image/ListDB":
		return pb.PERMISSION_IMAGE_INFO_READ
	case "/image.Image/Remove",
		"/image.Image/Update",
		"/image.Image/Upload",
		"/image.Image/Download":
		return pb.PERMISSION_IMAGE_INFO_WRITE
	case "/image.Image/Approve":
		return pb.PERMISSION_AUDIT_APPROVE_WRITE
	case "/logging.Logging/ListRuntime":
		return pb.PERMISSION_AUTID_LOG_READ
	case "/logging.Logging/ListWarn",
		"/logging.Logging/ReadWarn":
		return pb.PERMISSION_AUTID_WARN_READ
	default:
		return pb.PERMISSION_NONE
	}
}

func hasPerm(target pb.PERMISSION, perms []*pb.Permission) bool {
	for i := 0; i < len(perms); i++ {
		p := perms[i]
		if p != nil && p.Id == int32(target) && p.Allow {
			return true
		}
	}
	return false
}
