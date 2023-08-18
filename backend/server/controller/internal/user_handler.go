package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"ksc-mcube/model"
	"ksc-mcube/rpc"
	pb "ksc-mcube/rpc/pb/user"
	"ksc-mcube/server"
)

type UserServer struct {
	pb.UnimplementedUserServer
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
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

	return &pb.LoginReply{
		UserId:  userInfo.ID,
		AuthKey: fmt.Sprintf("%d%c%s", userInfo.ID, server.AuthKeySeprator, sessionKey),
	}, nil
}

func (s *UserServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Infof("get metadata from incoming context failed")
		return nil, rpc.ErrInternal
	}

	values := md["authorization"]
	if len(values) == 0 {
		log.Infof("'authorization' not exist in request metadata.")
		return nil, rpc.ErrInternal
	}

	authorization := values[0]
	i := strings.IndexRune(authorization, ':')
	if i == -1 {
		log.Infof("invalid authorization metadata: %v", authorization)
		return nil, rpc.ErrInternal
	}

	userID, accessToken := authorization[:i], authorization[i+1:]
	err := model.RemoveSession(userID, accessToken)
	if err != nil && err != model.ErrRecordNotFound {
		log.Infof("database error=%v", err)
		return nil, rpc.ErrInternal
	} else if err == model.ErrRecordNotFound {
		return nil, rpc.ErrNotFound
	}

	return &pb.LogoutReply{}, nil
}

func (s *UserServer) Signup(ctx context.Context, in *pb.SignupRequest) (*pb.SignupReply, error) {
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
	if err != nil && err != model.ErrRecordNotFound {
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
