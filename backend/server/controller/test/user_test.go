package test

import (
	"context"
	"testing"

	common "ksc-mcube/rpc/pb/common"
	pb "ksc-mcube/rpc/pb/user"

	"google.golang.org/grpc"
)

func TestUserSignup(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewUserClient(conn)
		request := pb.SignupRequest{
			Header:   &common.RequestHeader{},
			Username: "test",
			Password: "12345678",
			Role:     "test",
		}
		reply, err := cli.Signup(ctx, &request)
		if err != nil {
			t.Errorf("Signup: %v", err)
		}

		t.Logf("Signup reply: %v", reply)
	})

}

func TestUserLogin(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewUserClient(conn)
		request := pb.LoginRequest{
			Header:   &common.RequestHeader{},
			Username: "test",
			Password: "12345678",
		}

		reply, err := cli.Login(ctx, &request)
		if err != nil {
			t.Errorf("Login: %v", err)
		}

		t.Logf("Login reply: %v", reply)
	})
}
