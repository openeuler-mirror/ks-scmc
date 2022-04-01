package test

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	pb "scmc/rpc/pb/user"
)

func TestUserSignup(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewUserClient(conn)
		request := pb.SignupRequest{
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

func TestUserLogout(t *testing.T) {
	testRunner(func(ctx context.Context, conn *grpc.ClientConn) {
		cli := pb.NewUserClient(conn)
		request := pb.LogoutRequest{}

		reply, err := cli.Logout(ctx, &request)
		if err != nil {
			t.Errorf("Logout: %v", err)
		}

		t.Logf("Logout reply: %v", reply)
	})
}
