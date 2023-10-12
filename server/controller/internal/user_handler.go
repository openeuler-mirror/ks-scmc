package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"

	"scmc/model"
	"scmc/rpc"
	pb "scmc/rpc/pb/user"
	"scmc/server"
)

func generatePassword(input string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
}

func getUserFromContext(ctx context.Context) (string, string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Infof("get metadata from incoming context failed")
		return "", "", false
	}

	values := md["authorization"]
	if len(values) == 0 {
		log.Infof("'authorization' not exist in request metadata.")
		return "", "", false
	}

	authorization := values[0]
	i := strings.IndexRune(authorization, ':')
	if i == -1 {
		log.Infof("invalid authorization metadata: %v", authorization)
		return "", "", false
	}

	return authorization[:i], authorization[i+1:], true
}

type UserServer struct {
	pb.UnimplementedUserServer
}

func (s *UserServer) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginReply, error) {
	// check inputs
	if len(in.Username) < 4 || len(in.Password) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	// database operations
	userInfo, err := model.QueryUser(ctx, in.Username)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	} else if userInfo == nil {
		return nil, rpc.ErrNotFound
	}

	// check password
	if err := userInfo.CheckPassword(in.Password); err != nil {
		log.Infof("check password: %v", err)
		return nil, rpc.ErrWrongPassword
	}

	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		addr = pr.Addr.String()
	}

	// UUID create session
	sessionKey := uuid.New().String()
	if err := model.CreateSession(userInfo.ID, sessionKey, addr); err != nil {
		log.Warn("create seccion err: %v", err)
		return &pb.LoginReply{}, nil
	}

	return &pb.LoginReply{
		UserId:  userInfo.ID,
		AuthKey: fmt.Sprintf("%d%c%s", userInfo.ID, server.AuthKeySeprator, sessionKey),
	}, nil
}

func (s *UserServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	userID, accessToken, ok := getUserFromContext(ctx)
	if !ok {
		return nil, rpc.ErrInternal
	}

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
	if len(in.Username) < 4 || len(in.Password) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	// pre-process: bcrypt password
	rawBytes, err := generatePassword(in.Password)
	if err != nil {
		log.Warnf("generatePassword: %v", err)
		return nil, rpc.ErrInternal
	}

	// database operations
	userInfo, err := model.QueryUser(ctx, in.Username)
	if err != nil && err != model.ErrRecordNotFound {
		return nil, rpc.ErrInternal
	} else if userInfo != nil {
		return nil, rpc.ErrAlreadyExists
	}

	var newUserInfo *model.UserInfo = &model.UserInfo{
		Username:   in.Username,
		RealName:   in.Username,
		PasswordEn: string(rawBytes),
	}
	if err := model.CreateUser(ctx, newUserInfo); err != nil {
		return nil, rpc.ErrInternal
	}

	// retrive user info

	// finish
	return &pb.SignupReply{}, nil
}

func (s *UserServer) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	userID, accessToken, ok := getUserFromContext(ctx)
	if !ok {
		return nil, rpc.ErrInternal
	}

	if in.OldPassword == "" || len(in.NewPassword) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	userInfo, err := model.QueryUserByID(userID)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	} else if userInfo == nil {
		return nil, rpc.ErrNotFound
	}

	if err := userInfo.CheckPassword(in.OldPassword); err != nil {
		log.Infof("check password: %v", err)
		return nil, rpc.ErrWrongPassword
	}

	if in.NewPassword == in.OldPassword {
		log.Infof("new password identical to old one")
		return nil, rpc.ErrInvalidArgument
	}

	rawBytes, err := generatePassword(in.NewPassword)
	if err != nil {
		log.Warnf("generatePassword: %v", err)
		return nil, rpc.ErrInternal
	}

	if err := userInfo.UpdatePassword(string(rawBytes)); err != nil {
		return nil, rpc.ErrInternal
	}

	if err := model.RemoveSession(userID, accessToken); err != nil {
		log.Info("UpdatePassword user_id=%d, RemoveSession err=%v", userID, err)
	}

	return &pb.UpdatePasswordReply{NeedRelogin: true}, nil
}

// 用户列表
func (s *UserServer) ListUser(ctx context.Context, in *pb.ListUserRequest) (*pb.ListUserReply, error) {
	users, err := model.ListUser()
	if err != nil {
		log.Errorf("ListUser err=%v", err)
		return nil, rpc.ErrInternal
	}

	roles, err := model.ListRole()
	if err != nil {
		log.Errorf("ListRole err=%v", err)
		return nil, rpc.ErrInternal
	}

	rolesMap := make(map[int64]*model.UserRole, len(roles))
	for i := 0; i < len(roles); i++ {
		r := roles[i]
		rolesMap[r.ID] = r
	}

	reply := pb.ListUserReply{}
	for _, userinfo := range users {
		u := &pb.UserInfo{
			Id:         userinfo.ID,
			LoginName:  userinfo.Username,
			RealName:   userinfo.RealName,
			IsActive:   userinfo.IsActive,
			IsEditable: userinfo.IsEditable,
			RoleId:     userinfo.RoleID,
			CreatedAt:  userinfo.CreatedAt,
			UpdatedAt:  userinfo.UpdatedAt,
		}

		if r, ok := rolesMap[u.RoleId]; ok {
			u.RoleInfo = &pb.UserRole{
				Id:   r.ID,
				Name: r.Name,
			}
		}
		reply.Users = append(reply.Users, u)
	}

	return &reply, nil
}

// 创建新用户
func (s *UserServer) CreateUser(ctx context.Context, in *pb.CreateUserRequest) (*pb.CreateUserReply, error) {
	if in.UserInfo == nil {
		return nil, rpc.ErrInvalidArgument
	} else if len(in.UserInfo.LoginName) < 4 || len(in.UserInfo.Password) < 8 || in.UserInfo.RoleId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}
	roleInfo, err := model.QueryRoleById(ctx, in.UserInfo.RoleId)
	if err != nil {
		log.Warnf("QueryRoleById: %v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrInvalidArgument
		}
		return nil, rpc.ErrInternal
	}

	rawBytes, err := generatePassword(in.UserInfo.Password)
	if err != nil {
		log.Warnf("generatePassword: %v", err)
		return nil, rpc.ErrInternal
	}

	userInfo := &model.UserInfo{
		Username:   in.UserInfo.LoginName,
		RealName:   in.UserInfo.RealName,
		PasswordEn: string(rawBytes),
		IsActive:   in.UserInfo.IsActive,
		IsEditable: in.UserInfo.IsEditable,
		RoleID:     in.UserInfo.RoleId,
	}
	err = model.CreateUser(ctx, userInfo)
	if err != nil {
		log.Warnf("createUser: %v", err)
		if err == model.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		return nil, rpc.ErrInternal
	}

	var perms []*pb.Permission
	err = json.Unmarshal([]byte(roleInfo.PermsJSON), &perms)
	if err != nil {
		log.Warnf("json unmarshal err: %v", err)
	}
	return &pb.CreateUserReply{
		UserInfo: &pb.UserInfo{
			Id:         userInfo.ID,
			LoginName:  userInfo.Username,
			RealName:   userInfo.RealName,
			IsActive:   userInfo.IsActive,
			IsEditable: userInfo.IsEditable,
			RoleId:     userInfo.RoleID,
			CreatedAt:  userInfo.CreatedAt,
			UpdatedAt:  userInfo.UpdatedAt,
			RoleInfo: &pb.UserRole{
				Id:         roleInfo.ID,
				Name:       roleInfo.Name,
				IsEditable: roleInfo.IsEditable,
				CreatedAt:  roleInfo.CreatedAt,
				UpdatedAt:  roleInfo.UpdatedAt,
				Perms:      perms,
			},
		},
	}, nil
}

// 更新用户信息
func (s *UserServer) UpdateUser(ctx context.Context, in *pb.UpdateUserRequest) (*pb.UpdateUserReply, error) {
	if in.UserInfo == nil {
		return nil, rpc.ErrInvalidArgument
	} else if len(in.UserInfo.LoginName) < 4 || in.UserInfo.RoleId <= 0 {
		return nil, rpc.ErrInvalidArgument
	} else if len(in.UserInfo.Password) > 0 && len(in.UserInfo.Password) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	userInfo, err := model.QueryUserByID(in.UserInfo.Id)
	if err != nil {
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	} else if userInfo == nil {
		return nil, rpc.ErrNotFound
	}

	userInfo.IsActive = in.UserInfo.IsActive
	if len(in.UserInfo.RealName) > 0 {
		userInfo.RealName = in.UserInfo.RealName
	}
	if len(in.UserInfo.Password) > 0 {
		rawBytes, err := generatePassword(in.UserInfo.Password)
		if err != nil {
			log.Warnf("generatePassword: %v", err)
			return nil, rpc.ErrInternal
		}
		userInfo.PasswordEn = string(rawBytes)
	}
	var roleInfo *model.UserRole
	if in.UserInfo.RoleId > 0 {
		if roleInfo, err = model.QueryRoleById(ctx, in.UserInfo.RoleId); err != nil {
			log.Warnf("QueryRoleById: %v", err)
			if err == model.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			return nil, rpc.ErrInternal
		}
		userInfo.RoleID = in.UserInfo.RoleId
	}

	if err = model.UpdateUser(ctx, userInfo); err != nil {
		log.Warnf("updateUser: %v", err)
		if err == model.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	}

	var perms []*pb.Permission
	err = json.Unmarshal([]byte(roleInfo.PermsJSON), &perms)
	if err != nil {
		log.Warnf("json unmarshal err: %v", err)
	}
	return &pb.UpdateUserReply{
		UserInfo: &pb.UserInfo{
			Id:         userInfo.ID,
			LoginName:  userInfo.Username,
			RealName:   userInfo.RealName,
			IsActive:   userInfo.IsActive,
			IsEditable: userInfo.IsEditable,
			RoleId:     userInfo.RoleID,
			CreatedAt:  userInfo.CreatedAt,
			UpdatedAt:  userInfo.UpdatedAt,
			RoleInfo: &pb.UserRole{
				Id:         roleInfo.ID,
				Name:       roleInfo.Name,
				IsEditable: roleInfo.IsEditable,
				CreatedAt:  roleInfo.CreatedAt,
				UpdatedAt:  roleInfo.UpdatedAt,
				Perms:      perms,
			},
		},
	}, nil

}

// 删除用户
func (s *UserServer) RemoveUser(ctx context.Context, in *pb.RemoveUserRequest) (*pb.RemoveUserReply, error) {
	if len(in.UserIds) <= 0 {
		log.Info("RemoveUser no input user_ids")
		return nil, rpc.ErrInvalidArgument
	}

	for _, i := range in.UserIds {
		if i <= 0 {
			log.Infof("RemoveUser get invalid user_id=%v", i)
			return nil, rpc.ErrInvalidArgument
		}
	}

	users, err := model.QueryUsers(in.UserIds)
	if err != nil {
		log.Infof("QueryUsers err=%v", err)
		return nil, rpc.ErrInternal
	}

	for _, u := range users {
		if !u.IsEditable {
			log.Infof("RemoveUser user_id=%v not editable", u.ID)
			return nil, rpc.ErrInvalidArgument
		}
	}

	err = model.RemoveUsers(users)
	if err != nil {
		log.Infof("RemoveUser user_ids=%v db err=%v", in.UserIds, err)
		return nil, rpc.ErrInternal
	}

	return &pb.RemoveUserReply{}, nil
}

// 用户角色列表
func (s *UserServer) ListRole(ctx context.Context, in *pb.ListRoleRequest) (*pb.ListRoleReply, error) {
	roles, err := model.ListRole()

	if err != nil {
		log.Errorf("ListUser err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListRoleReply
	for _, r := range roles {
		var perms []*pb.Permission
		json.Unmarshal([]byte(r.PermsJSON), &perms)

		reply.Roles = append(reply.Roles, &pb.UserRole{
			Id:         r.ID,
			Name:       r.Name,
			IsEditable: r.IsEditable,
			Perms:      perms,
			CreatedAt:  r.CreatedAt,
			UpdatedAt:  r.UpdatedAt,
		})
	}

	return &reply, nil

}

// 创建新角色
func (s *UserServer) CreateRole(ctx context.Context, in *pb.CreateRoleRequest) (*pb.CreateRoleReply, error) {
	if in.RoleInfo == nil {
		return nil, rpc.ErrInvalidArgument
	} else if in.RoleInfo.Id < 0 || len(in.RoleInfo.Name) < 4 || len(in.RoleInfo.Perms) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	permsJSON, err := json.Marshal(in.RoleInfo.Perms)
	if err != nil {
		log.Warnf("marshal perms json err=%v", err)
		return nil, rpc.ErrInternal
	}

	role := &model.UserRole{
		Name:       in.RoleInfo.Name,
		IsEditable: in.RoleInfo.IsEditable,
		PermsJSON:  string(permsJSON),
	}

	if err = model.CreateRole(ctx, role); err != nil {
		log.Warnf("createUser: %v", err)
		if err == model.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		return nil, rpc.ErrInternal
	}

	var perms []*pb.Permission
	err = json.Unmarshal([]byte(role.PermsJSON), &perms)
	if err != nil {
		log.Warnf("json unmarshal err: %v", err)
	}
	return &pb.CreateRoleReply{
		RoleInfo: &pb.UserRole{
			Id:         role.ID,
			Name:       role.Name,
			IsEditable: role.IsEditable,
			CreatedAt:  role.CreatedAt,
			UpdatedAt:  role.UpdatedAt,
			Perms:      perms,
		},
	}, nil
}

// 更新角色信息
func (s *UserServer) UpdateRole(ctx context.Context, in *pb.UpdateRoleRequest) (*pb.UpdateRoleReply, error) {
	if in.RoleInfo == nil {
		return nil, rpc.ErrInvalidArgument
	} else if in.RoleInfo.Id < 0 || len(in.RoleInfo.Name) < 4 || len(in.RoleInfo.Perms) == 0 {
		return nil, rpc.ErrInvalidArgument
	}

	role, err := model.QueryRoleById(ctx, in.RoleInfo.Id)
	if err != nil && err != model.ErrRecordNotFound {
		log.Warnf("db query role: %v", err)
		return nil, rpc.ErrInternal
	} else if err == model.ErrRecordNotFound {
		return nil, rpc.ErrNotFound
	}

	permsJSON, err := json.Marshal(in.RoleInfo.Perms)
	if err != nil {
		log.Warnf("marshal perms json err=%v", err)
		return nil, rpc.ErrInternal
	}

	role.Name = in.RoleInfo.Name
	role.PermsJSON = string(permsJSON)

	if err = model.UpdateRole(ctx, role); err != nil {
		log.Warnf("role UpdateRole err: %v", err)
		return nil, err
	}

	var perms []*pb.Permission
	err = json.Unmarshal([]byte(role.PermsJSON), &perms)
	if err != nil {
		log.Warnf("json unmarshal err: %v", err)
	}
	return &pb.UpdateRoleReply{
		RoleInfo: &pb.UserRole{
			Id:         role.ID,
			Name:       role.Name,
			IsEditable: role.IsEditable,
			CreatedAt:  role.CreatedAt,
			UpdatedAt:  role.UpdatedAt,
			Perms:      perms,
		},
	}, nil
}

// 删除角色
func (s *UserServer) RemoveRole(ctx context.Context, in *pb.RemoveRoleRequest) (*pb.RemoveRoleReply, error) {
	if in.RoleId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	err := model.RemoveRole(ctx, in.RoleId)
	if err != nil {
		log.Warnf("db remove role: %v", err)
		return nil, err
	}
	return &pb.RemoveRoleReply{}, nil
}
