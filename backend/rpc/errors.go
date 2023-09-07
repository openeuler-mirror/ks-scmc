package rpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "scmc/rpc/pb/common"
)

var (
	ErrOK                 = error(nil)
	ErrCanceled           = status.Error(codes.Canceled, "请求取消")
	ErrUnknown            = status.Error(codes.Unknown, "未知错误")
	ErrInvalidArgument    = status.Error(codes.InvalidArgument, "参数错误")
	ErrDeadlineExceeded   = status.Error(codes.DeadlineExceeded, "请求超时")
	ErrNotFound           = status.Error(codes.NotFound, "资源不存在")
	ErrAlreadyExists      = status.Error(codes.AlreadyExists, "资源冲突")
	ErrPermissionDenied   = status.Error(codes.PermissionDenied, "拒绝请求")
	ErrResourceExhausted  = status.Error(codes.ResourceExhausted, "") // framework error
	ErrFailedPrecondition = status.Error(codes.FailedPrecondition, "unknown")
	ErrAborted            = status.Error(codes.Aborted, "unknown")
	ErrOutOfRange         = status.Error(codes.OutOfRange, "unknown")
	ErrUnimplemented      = status.Error(codes.Unimplemented, "") // framework error
	ErrInternal           = status.Error(codes.Internal, "内部错误")  // framework error | rpc failure
	ErrUnavailable        = status.Error(codes.Unavailable, "")   // framework error
	ErrDataLoss           = status.Error(codes.DataLoss, "unknown")
	ErrUnauthenticated    = status.Error(codes.Unauthenticated, "用户请求未认证")
	ErrWrongPassword      = status.Error(codes.Code(pb.Errno_WrongPassword), "密码错误")
	ErrSomeConfigFailed   = status.Error(codes.Internal, "容器已创建, 部分配置失败")
	ErrInvalidImage       = status.Error(codes.Internal, "无效的镜像")

// ErrUserNotExist
)
