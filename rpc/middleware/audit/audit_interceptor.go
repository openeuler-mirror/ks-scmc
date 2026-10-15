package audit

import (
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"scmc/model/db"
	"scmc/rpc/middleware/auth"
	"scmc/rpc/pb/logging"
)

func rpcNeedLog(method string) bool {
	switch method {
	case "/user.User/Login",
		"/user.User/Logout",
		"/user.User/UpdatePassword",
		"/user.User/CreateUser",
		"/user.User/RemoveUser",
		"/user.User/UpdateUser",
		"/user.User/CreateRole",
		"/user.User/RemoveRole",
		"/user.User/UpdateRole",
		"/node.Node/Create",
		"/node.Node/Update",
		"/node.Node/Remove",
		"/container.Container/Create",
		"/container.Container/Start",
		"/container.Container/Stop",
		"/container.Container/Remove",
		"/container.Container/Restart",
		// "/container.Container/",
		// "/container.Container/",
		"/image.Image/Upload",
		"/image.Image/Download",
		"/image.Image/Approve",
		"/image.Image/Remove":
		return true
	}
	return false
}

func UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		if rpcNeedLog(info.FullMethod) {
			defer func(ctx context.Context) {
				uid, ok := auth.UserIDFromContext(ctx)
				if !ok {
					log.Warnf("UserIDFromContext faild ctx: %++v", ctx)
				}

				writeLog(uid, req.(proto.Message), err)

			}(ctx)
		}
		return handler(ctx, req)
	}
}

func StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := stream.Context()
		wrapperStream := NewWrapperStream(stream, ctx)

		defer func(ctx context.Context) {
			uid, ok := auth.UserIDFromContext(ctx)
			if !ok {
				log.Warnf("UserIDFromContext faild ctx: %++v", ctx)
			}

			msg := wrapperStream.Context().Value("message")
			writeLog(uid, msg.(proto.Message), err)
		}(ctx)

		return handler(srv, wrapperStream)
	}
}

func writeLog(uid int64, v interface{}, err error) {
	if v == nil {
		log.Infof("writeLog  msg=%+v", v)
		return
	}

	logData := parser{}.Parse(v)
	if logData != nil {
		logData.UserID = uid
		if s, _ := status.FromError(err); s != nil {
			logData.StatusCode = int64(s.Code())
			logData.Error = s.Message()
		}

		if t, ok := logging.EVENT_TYPE_name[int32(logData.EventType)]; ok {
			logData.EventType_ = t
		}
		data := []*db.RuntimeLog{logData}
		if err = db.CreateRuntimeLog(data); err != nil {
			log.Warnf("writeLog data=%+v err=%v", logData, err)
		}
	}
}
