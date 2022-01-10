package internal

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/peer"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/user"
)

type UserServer struct {
	pb.UnimplementedUserServer
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	// log.Infof("Received: %v", in)

	// check inputs
	if len(in.Username) < 4 || len(in.Password) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	// database operations
	userInfo, err := model.QueryUser(in.Username)
	if err != nil {
		return &pb.LoginReply{}, nil
	} else if userInfo == nil {
		return nil, rpc.ErrNotFound
	}

	// check password
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.PasswordEn), []byte(in.Password)); err != nil {
		log.Infof("compare password: %v", err)
		return nil, rpc.ErrWrongPassword
	}

	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		addr = pr.Addr.String()
	}

	// UUID create session
	sessionKey := uuid.New().String()
	if err := model.CreateSession(userInfo.ID, sessionKey, addr); err != nil {
		return &pb.LoginReply{}, nil
	}

	return &pb.LoginReply{UserId: userInfo.ID, SessionKey: sessionKey}, nil
}

func (s *UserServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	log.Infof("Received: %v", in)

	// TODO database operation
	return &pb.LogoutReply{}, nil
}

func (s *UserServer) Signup(ctx context.Context, in *pb.SignupRequest) (*pb.SignupReply, error) {
	// log.Infof("Received: %v", in)

	// check inputs, TODO(check character set)
	if len(in.Username) < 4 || len(in.Password) < 8 || len(in.Role) < 4 {
		return nil, rpc.ErrInvalidArgument
	}

	// pre-process: bcrypt password
	rawBytes, err := bcrypt.GenerateFromPassword([]byte(in.GetPassword()), 14)
	if err != nil {
		log.Warnf("bcrypt handle password: %v", err)
		return nil, rpc.ErrInternal
	}

	// database operations
	userInfo, err := model.QueryUser(in.Username)
	if err != nil {
		return nil, rpc.ErrInternal
	} else if userInfo != nil {
		return nil, rpc.ErrAlreadyExists
	}

	if err := model.CreateUser(in.Username, string(rawBytes), in.Role); err != nil {
		return nil, rpc.ErrInternal
	}

	// retrive user info

	// finish
	return &pb.SignupReply{}, nil
}
