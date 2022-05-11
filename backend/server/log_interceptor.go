package server

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"scmc/model"
	"scmc/rpc/pb/container"
	"scmc/rpc/pb/image"
	"scmc/rpc/pb/logging"
	"scmc/rpc/pb/node"
	"scmc/rpc/pb/user"
)

// LogInterceptor is a server interceptor for request and reply logging
type LogInterceptor struct {
	needRuntimeLog bool
}

// NewLogInterceptor returns a new log interceptor
func NewLogInterceptor(needRuntimeLog bool) *LogInterceptor {
	return &LogInterceptor{needRuntimeLog}
}

// Unary returns a server interceptor function to logging unary RPC
func (i *LogInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (reply interface{}, err error) {
		ts := time.Now()

		var addr string
		if p, ok := peer.FromContext(ctx); ok {
			addr = p.Addr.String()
		}

		log.Debugf("%s %s\nREQUEST: %+v", addr, info.FullMethod, req)

		defer func() {
			if err != nil {
				log.Infof("%s %s\nREQUEST: %+v ERR: %v", addr, info.FullMethod, req, err)
			}

			duration := time.Since(ts)
			if duration.Seconds() > 0.999 {
				log.Infof("%s %s COST: %v", addr, info.FullMethod, duration)
			} else {
				log.Debugf("%s %s\nREQUEST: %+v\nREPLY: %+v\nCOST: %v", addr, info.FullMethod, req, reply, duration)
			}

			if i.needRuntimeLog {
				writeRuntimeLog(ctx, req.(proto.Message), err)
			}

		}()

		reply, err = handler(ctx, req)
		return reply, err
	}
}

func (interceptor *LogInterceptor) Streams() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, newWrappedStream(ss))
		return err
	}
}

type wrappedStream struct {
	// context      context.Context
	// method       string
	grpc.ServerStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	logger := log.Debugf
	logger("Receive a message (Type: %T) at %s", m, time.Now().Format(time.RFC3339))
	var err error
	defer func() {
		err = w.ServerStream.RecvMsg(m)
		writeRuntimeLog(w.Context(), m.(proto.Message), err)
	}()
	return err
}

func (w *wrappedStream) SendMsg(m interface{}) error {
	logger := log.Debugf
	var err error
	defer func() {
		err = w.ServerStream.SendMsg(m)
		writeRuntimeLog(w.Context(), m.(proto.Message), err)
	}()
	logger("Send a message (Type: %T) at %v", m, time.Now().Format(time.RFC3339))
	return err
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

func writeRuntimeLog(ctx context.Context, req interface{}, err error) {
	// 获取执行用户
	var userID int64
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		if values, ok := md["authorization"]; ok {
			authorization := values[0]
			i := strings.IndexRune(authorization, AuthKeySeprator)
			if i == -1 {
				log.Infof("invalid authorization metadata: %v", authorization)
			} else {
				s := authorization[:i]
				i, err := strconv.Atoi(s)
				if err != nil {
					log.Warnf("parse userid auth=%v str=%v", authorization, s)
				}
				userID = int64(i)
			}
		}
	}

	var logData *model.RuntimeLog
	switch reqMsg := req.(type) {
	case *user.LoginRequest:
		logData = RuntimeLogWritter{}.Login(reqMsg)
	case *user.LogoutRequest:
		logData = RuntimeLogWritter{}.Logout(reqMsg)
	case *user.UpdatePasswordRequest:
		logData = RuntimeLogWritter{}.UpdatePassword(reqMsg)
	case *user.CreateUserRequest:
		logData = RuntimeLogWritter{}.CreateUser(reqMsg)
	case *user.RemoveUserRequest:
		logData = RuntimeLogWritter{}.RemoveUser(reqMsg)
	case *user.UpdateUserRequest:
		logData = RuntimeLogWritter{}.UpdateUser(reqMsg)
	case *user.CreateRoleRequest:
		logData = RuntimeLogWritter{}.CreateRole(reqMsg)
	case *user.RemoveRoleRequest:
		logData = RuntimeLogWritter{}.RemoveRole(reqMsg)
	case *user.UpdateRoleRequest:
		logData = RuntimeLogWritter{}.UpdateRole(reqMsg)
	case *node.CreateRequest:
		logData = RuntimeLogWritter{}.CreateNode(reqMsg)
	case *node.UpdateRequest:
		logData = RuntimeLogWritter{}.UpdateNode(reqMsg)
	case *node.RemoveRequest:
		logData = RuntimeLogWritter{}.RemoveNode(reqMsg)
	case *container.RemoveRequest:
		logData = RuntimeLogWritter{}.RemoveContainer(reqMsg)
	case *container.RestartRequest:
		logData = RuntimeLogWritter{}.RestartContainer(reqMsg)
	case *container.CreateRequest:
		logData = RuntimeLogWritter{}.CreateContainer(reqMsg)
	case *container.StartRequest:
		logData = RuntimeLogWritter{}.StartContainer(reqMsg)
	case *container.StopRequest:
		logData = RuntimeLogWritter{}.StopContainer(reqMsg)
	case *image.RemoveRequest:
		logData = RuntimeLogWritter{}.RemoveImage(reqMsg)
	case *image.ApproveRequest:
		logData = RuntimeLogWritter{}.ApproveImage(reqMsg)
	case *image.UploadRequest:
		logData = RuntimeLogWritter{}.UploadImage(reqMsg)
	case *image.DownloadRequest:
		logData = RuntimeLogWritter{}.DownloadImage(reqMsg)
	case nil:
		log.Warn("nil")
	case *container.InspectRequest, *container.ListRequest, *container.ListTemplateRequest:
	case *node.ListRequest, *node.StatusRequest:
	case *image.ListDBRequest, *image.ListRequest:
		log.Debugf("ignore message type=%T", reqMsg)
		return
	default:
		log.Infof("message not recogonized, type=%T", reqMsg)
		return
	}

	if logData != nil {
		logData.UserID = userID
		if s, _ := status.FromError(err); s != nil {
			logData.StatusCode = int64(s.Code())
			logData.Error = s.Message()
		}

		if t, ok := logging.EVENT_TYPE_name[int32(logData.EventType)]; ok {
			logData.EventType_ = t
		}
		data := []*model.RuntimeLog{logData}
		if err = model.CreateRuntimeLog(data); err != nil {
			log.Warnf("CreateLog %+v err=%v", logData, err)
		}
	}
}
