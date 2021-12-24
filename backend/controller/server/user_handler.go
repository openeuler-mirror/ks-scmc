package server

import (
	"context"
	"log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/peer"

	"ksc-mcube/model"
	"ksc-mcube/rpc/errno"
	pb "ksc-mcube/rpc/pb/user"
)

type UserServer struct {
	pb.UnimplementedUserServer
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	log.Printf("Received: %v", in)

	// check inputs
	if len(in.Username) < 4 || len(in.Password) < 8 {
		return &pb.LoginReply{Header: errno.InvalidArgument}, nil
	}

	// database operations
	userInfo, err := model.QueryUser(in.Username)
	if err != nil {
		return &pb.LoginReply{Header: errno.InternalError}, nil
	} else if userInfo == nil {
		return &pb.LoginReply{Header: errno.UserNotExist}, nil
	}

	// check password
	if err := bcrypt.CompareHashAndPassword([]byte(userInfo.PasswordEn), []byte(in.Password)); err != nil {
		log.Printf("compare password: %v", err)
		return &pb.LoginReply{Header: errno.WrongPassword}, nil
	}

	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		addr = pr.Addr.String()
	}

	// UUID create session
	sessionKey := uuid.New().String()
	if err := model.CreateSession(userInfo.ID, sessionKey, addr); err != nil {
		return &pb.LoginReply{Header: errno.InternalError}, nil
	}

	return &pb.LoginReply{Header: errno.OK, UserId: userInfo.ID, SessionKey: sessionKey}, nil
}

func (s *UserServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	log.Printf("Received: %v", in)
	return &pb.LogoutReply{Header: errno.OK}, nil
}

func (s *UserServer) Signup(ctx context.Context, in *pb.SignupRequest) (*pb.SignupReply, error) {
	log.Printf("Received: %v", in)

	// check inputs, TODO(check character set)
	if len(in.Username) < 4 || len(in.Password) < 8 || len(in.Role) < 4 {
		return &pb.SignupReply{Header: errno.InvalidArgument}, nil
	}

	// pre-process: bcrypt password
	rawBytes, err := bcrypt.GenerateFromPassword([]byte(in.GetPassword()), 14)
	if err != nil {
		log.Printf("bcrypt handle password: %v", err)
		return &pb.SignupReply{Header: errno.InternalError}, nil
	}

	// database operations
	userInfo, err := model.QueryUser(in.Username)
	if err != nil {
		return &pb.SignupReply{Header: errno.InternalError}, nil
	} else if userInfo != nil {
		return &pb.SignupReply{Header: errno.UserAlreadyExist}, nil
	}

	if err := model.CreateUser(in.Username, string(rawBytes), in.Role); err != nil {
		return &pb.SignupReply{Header: errno.InternalError}, nil
	}

	// retrive user info

	// finish
	return &pb.SignupReply{Header: errno.OK}, nil
}
