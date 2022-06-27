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
			if err != nil && common.NeedCheckAuth() {
				return nil, err
			}

			if common.NeedCheckAuth() {
				if err := auth.checkAuthentication(userID, accessToken); err != nil {
					// log.Infof("AuthInterceptor error: %v", e)
					return nil, err
				}
				if info.FullMethod != "/user.User/Logout" && info.FullMethod != "/user.User/UpdatePassword" && common.NeedCheckPerm() {
					ctx, err = auth.checkAuthorization(ctx, userID, info.FullMethod)
					if err != nil {
						log.Infof("AuthInterceptor method=%v user=%v error=%v", info.FullMethod, userID, err)
						return nil, err
					}
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

			if common.NeedCheckAuth() {
				if err := auth.checkAuthentication(userID, accessToken); err != nil {
					// log.Infof("AuthInterceptor error: %v", e)
					return err
				}
			}

			if common.NeedCheckPerm() {
				if _, err = auth.checkAuthorization(context.Background(), userID, info.FullMethod); err != nil {
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

func (auth *AuthInterceptor) checkAuthorization(ctx context.Context, userID, method string) (context.Context, error) {
	permID := getPermID(method)
	if permID == 0 {
		log.Warnf("cannot recognize perm for method=%s", method)
		return nil, rpc.ErrPermissionDenied
	}

	userRole, err := model.QueryUserRole(userID)
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

	if !HasPerm(permID, perms) {
		return nil, rpc.ErrPermissionDenied
	}
	return context.WithValue(ctx, "PERMS", perms), nil
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

func HasPerm(target pb.PERMISSION, perms []*pb.Permission) bool {
	for i := 0; i < len(perms); i++ {
		p := perms[i]
		if p != nil && p.Id == int32(target) && p.Allow {
			return true
		}
	}
	return false
}
