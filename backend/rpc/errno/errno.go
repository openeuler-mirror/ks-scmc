package errno

import (
	pb "ksc-mcube/rpc/pb/common"
)

var (
	OK               = newHeader(pb.Errno_OK, "")
	Unknown          = newHeader(pb.Errno_Unknown, "未知错误")
	Unimplemented    = newHeader(pb.Errno_Unimplemented, "方法未实现")
	InternalError    = newHeader(pb.Errno_Internal, "服务器内部错误")
	InvalidArgument  = newHeader(pb.Errno_InvalidArgument, "参数错误")
	PermissionDenied = newHeader(pb.Errno_PermissionDenied, "拒绝请求")
	NotFound         = newHeader(pb.Errno_NotFound, "资源不存在")
	AlreadyExists    = newHeader(pb.Errno_AlreadyExists, "资源已存在")

	UserAlreadyExist = newHeader(pb.Errno_UserAlreadyExist, "")
	UserNotExist     = newHeader(pb.Errno_UserNotExist, "")
	WrongPassword    = newHeader(pb.Errno_WrongPassword, "")
	InvalidSession   = newHeader(pb.Errno_InvalidSession, "无效用户会话")

	/*
		OK = 0
		Canceled = 1
		Unknown = 2
		InvalidArgument = 3
		DeadlineExceeded = 4
		NotFound = 5
		AlreadyExists = 6
		PermissionDenied = 7
		ResourceExhausted = 8
		FailedPrecondition = 9
		Aborted = 10
		OutOfRange = 11
		Unimplemented = 12
		Internal = 13
		Unavailable = 14
		DataLoss = 15
		Unauthenticated = 16
	*/
)

func newHeader(errno pb.Errno, msg string) *pb.ReplyHeader {
	return &pb.ReplyHeader{ErrNo: int32(errno), ErrMsg: msg}
}
