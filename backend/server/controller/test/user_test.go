package test

import (
	"context"
	"fmt"
	"log"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

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

func TestListUser(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.ListUserRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.ListUserReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.ListUserRequest{},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cli.ListUser(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, user := range got.GetUsers() {
				log.Println(user)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.UpdateUserRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UpdateUserReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.UpdateUserRequest{
					UserInfo: &pb.UserInfo{
						Id:        1000,
						LoginName: "sean1000",
						RealName:  "sean.x",
						Password:  "password",
						RoleId:    1,
					},
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.UpdateUserRequest{
					UserInfo: &pb.UserInfo{
						Id:         33,
						LoginName:  "aean",
						RealName:   "sean",
						Password:   "password",
						IsActive:   true,
						IsEditable: true,
						RoleId:     84,
					},
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.UpdateUser(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}
func TestCreateUser(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.CreateUserRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.CreateUserReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.CreateUserRequest{
					UserInfo: &pb.UserInfo{
						LoginName:  "sean",
						RealName:   "sean",
						Password:   "pass12345",
						IsActive:   true,
						IsEditable: true,
						RoleId:     68,
					},
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.CreateUserRequest{
					UserInfo: &pb.UserInfo{
						LoginName: "seanaaa02",
						Password:  "passaa",
						RoleId:    3,
					},
				},
			},
		},
		{
			name: "test03",
			args: args{
				ctx: ctx,
				req: &pb.CreateUserRequest{
					UserInfo: &pb.UserInfo{
						Id:        133,
						LoginName: "aean03",
						RoleId:    1,
					},
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.CreateUser(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestRemoveUser(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.RemoveUserRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.RemoveUserRequest
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.RemoveUserRequest{
					//					UserId: 1018,
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.RemoveUserRequest{
					//					UserId: 10184,
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.RemoveUser(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

const (
	authkey   = "authorization"
	authvalue = "1000:9483e860-b44b-483b-90f5-4052038056e9"
)

func TestCreateRole(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.CreateRoleRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.CreateUserReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.CreateRoleRequest{
					RoleInfo: &pb.UserRole{
						Name: "role023",
					},
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.CreateRoleRequest{
					RoleInfo: &pb.UserRole{
						Name: "role04",
						Perms: []*pb.Permission{
							{Id: 1, Allow: true},
						},
					},
				},
			},
		},
		{
			name: "test03",
			args: args{
				ctx: ctx,
				req: &pb.CreateRoleRequest{
					RoleInfo: &pb.UserRole{
						Name: "test03",
						Perms: []*pb.Permission{
							{Id: 1, Allow: true},
						},
					},
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.CreateRole(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestUpdateRole(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}

	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.UpdateRoleRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.UpdateRoleReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.UpdateRoleRequest{
					RoleInfo: &pb.UserRole{
						Id:   67,
						Name: "role67",
					},
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.UpdateRoleRequest{
					RoleInfo: &pb.UserRole{
						Id:   84,
						Name: "admin",
						Perms: []*pb.Permission{
							{Id: 1, Allow: true},
						},
					},
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var header, trailer metadata.MD
			_, err := cli.UpdateRole(tt.args.ctx, tt.args.req, grpc.Header(&header), grpc.Trailer(&trailer))
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Println(header)
			fmt.Println(trailer)
		})
	}
}

func TestRemoveRole(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)
	type args struct {
		ctx context.Context
		req *pb.RemoveRoleRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.RemoveRoleReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.RemoveRoleRequest{
					RoleId: 67,
				},
			},
		},
		{
			name: "test02",
			args: args{
				ctx: ctx,
				req: &pb.RemoveRoleRequest{
					RoleId: 5,
				},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cli.RemoveRole(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

		})
	}
}

func TestListRole(t *testing.T) {
	ctx, conn, err := initTestRunner()
	if err != nil {
		log.Fatalln(err)
	}
	ctx = metadata.AppendToOutgoingContext(ctx, authkey, authvalue)

	type args struct {
		ctx context.Context
		req *pb.ListRoleRequest
	}
	tests := []struct {
		name    string
		args    args
		want    *pb.ListRoleReply
		wantErr bool
	}{
		{
			name: "test01",
			args: args{
				ctx: ctx,
				req: &pb.ListRoleRequest{},
			},
		},
	}

	cli := pb.NewUserClient(conn)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cli.ListRole(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// fmt.Printf("got.CurPage: %v, got.PerPage: %v, got.TotalPages: %v, got.TotalRows: %v\n",
			// 	got.CurPage, got.PerPage, got.TotalPages, got.TotalRows)
			for _, record := range got.Roles {
				fmt.Printf("id: %v, config: %v\n", record.Id, record.Name)
			}
		})
	}
}
