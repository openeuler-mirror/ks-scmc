package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"scmc/common"
	"scmc/model/db"
	"scmc/model/session"
	"scmc/rpc"
	"scmc/rpc/middleware/auth"
	pb "scmc/rpc/pb/user"
)

func generatePassword(input string) (string, error) {
	raw, err := bcrypt.GenerateFromPassword([]byte(input), bcrypt.DefaultCost)
	if err != nil {
		log.Warnf("bcrypt.GenerateFromPassword err=%v", err)
		return "", err
	}
	return string(raw), nil
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
	u, err := db.QueryUser(ctx, in.Username)
	if err != nil {
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
	} else if u == nil {
		return nil, rpc.ErrNotFound
	}

	lockRemain := time.Until(time.Unix(u.LockUntil, 0))
	if lockRemain > 0 {
		return nil, status.Errorf(codes.Unauthenticated, "账户已锁定，%v后重试", lockRemain.Round(time.Second))
	}

	// check password
	if err := u.CheckPassword(in.Password); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			if err := u.TryLock(); err != nil {
				log.Errorf("try lock user=%s: %v", u.Username, err)
			}
			time.Sleep(time.Second * 5)
			return nil, rpc.ErrWrongPassword
		}
		log.Errorf("check password for user=%s: %v", u.Username, err)
		return nil, rpc.ErrInternal
	}

	var addr string
	if pr, ok := peer.FromContext(ctx); ok {
		addr = pr.Addr.String()
	}

	// get user role info
	role, err := db.QueryRoleById(ctx, u.RoleID)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// UUID create session
	sessionKey := uuid.New().String()
	old, err := db.CreateSession(u, sessionKey, addr)
	if err != nil {
		log.Error("create session: ", err)
		return nil, rpc.ErrDatabaseFail
	}

	if old != nil {
		session.Timers.TriggerAndRemove(old.SessionKey)
	}

	authKey := fmt.Sprintf("%d%c%s", u.ID, common.UserSessionySeprator, sessionKey)

	// set authorizatio key in headers, for client interceptors' convenience
	grpc.SetHeader(ctx, metadata.Pairs("authorization", authKey))

	return &pb.LoginReply{
		UserId:  u.ID,
		AuthKey: authKey,
		UserRole: &pb.UserRole{
			Id:         role.ID,
			Name:       role.Name,
			IsEditable: role.IsEditable,
			CreatedAt:  role.CreatedAt,
			UpdatedAt:  role.UpdatedAt,
			Perms:      convertPerms(role.PermsJSON),
		},
	}, nil
}

func (s *UserServer) Logout(ctx context.Context, in *pb.LogoutRequest) (*pb.LogoutReply, error) {
	userID, sessionKey, ok := auth.UserSessionFromContext(ctx)
	if !ok {
		log.Warnf("context no user session, ctx=%+v", ctx)
		return nil, rpc.ErrInternal
	}

	err := db.RemoveSession(userID, sessionKey)
	if err != nil && err != db.ErrRecordNotFound {
		log.Infof("database error=%v", err)
		return nil, rpc.ErrInternal
	} else if err == db.ErrRecordNotFound {
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
	passwordEn, err := generatePassword(in.Password)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	// database operations
	userInfo, err := db.QueryUser(ctx, in.Username)
	if err != nil && err != db.ErrRecordNotFound {
		return nil, rpc.ErrInternal
	} else if userInfo != nil {
		return nil, rpc.ErrAlreadyExists
	}

	var newUserInfo *db.UserInfo = &db.UserInfo{
		Username:   in.Username,
		RealName:   in.Username,
		PasswordEn: passwordEn,
	}
	if err := db.CreateUser(ctx, newUserInfo); err != nil {
		return nil, rpc.ErrInternal
	}

	// retrive user info

	// finish
	return &pb.SignupReply{}, nil
}

func (s *UserServer) UpdatePassword(ctx context.Context, in *pb.UpdatePasswordRequest) (*pb.UpdatePasswordReply, error) {
	userID, session, ok := auth.UserSessionFromContext(ctx)
	if !ok {
		log.Warnf("context no user session, ctx=%+v", ctx)
		return nil, rpc.ErrInternal
	}

	u, err := db.QueryUserByID(userID)
	if err != nil {
		log.Warnf("query user_id=%v err=%v", userID, err)
		return nil, rpc.ErrDatabaseFail
	} else if u == nil {
		log.Warnf("UpdatePassword user_id=%v not exist", userID)
		return nil, rpc.ErrInternal
	}

	if in.OldPassword == "" || len(in.NewPassword) < 8 {
		return nil, rpc.ErrInvalidArgument
	}

	if err := u.CheckPassword(in.OldPassword); err != nil {
		log.Infof("check password: %v", err)
		return nil, rpc.ErrWrongPassword
	}

	if in.NewPassword == in.OldPassword {
		log.Debugf("new password identical to old one")
		return nil, status.Error(codes.InvalidArgument, "新密码无变化") // rpc.ErrInvalidArgument
	}

	passwordEn, err := generatePassword(in.NewPassword)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	if err := u.UpdatePassword(passwordEn); err != nil {
		return nil, rpc.ErrInternal
	}

	if err := db.RemoveSession(u.ID, session); err != nil {
		log.Infof("UpdatePassword user_id=%d, RemoveSession err=%v", u.ID, err)
	}

	return &pb.UpdatePasswordReply{NeedRelogin: true}, nil
}

// 用户列表
func (s *UserServer) ListUser(ctx context.Context, in *pb.ListUserRequest) (*pb.ListUserReply, error) {
	users, err := db.ListUser()
	if err != nil {
		log.Errorf("ListUser err=%v", err)
		return nil, rpc.ErrInternal
	}

	roles, err := db.ListRole()
	if err != nil {
		log.Errorf("ListRole err=%v", err)
		return nil, rpc.ErrInternal
	}

	rolesMap := make(map[int64]*db.UserRole, len(roles))
	for i := 0; i < len(roles); i++ {
		r := roles[i]
		rolesMap[r.ID] = r
	}

	var reply pb.ListUserReply
	for _, u := range users {
		user := &pb.UserInfo{
			Id:         u.ID,
			LoginName:  u.Username,
			RealName:   u.RealName,
			IsActive:   u.IsActive,
			IsEditable: u.IsEditable,
			RoleId:     u.RoleID,
			CreatedAt:  u.CreatedAt,
			UpdatedAt:  u.UpdatedAt,
		}

		if r, ok := rolesMap[user.RoleId]; ok {
			user.RoleInfo = &pb.UserRole{
				Id:   r.ID,
				Name: r.Name,
			}
		}
		reply.Users = append(reply.Users, user)
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
	roleInfo, err := db.QueryRoleById(ctx, in.UserInfo.RoleId)
	if err != nil {
		log.Warnf("QueryRoleById: %v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrInvalidArgument
		}
		return nil, rpc.ErrInternal
	}

	passwordEn, err := generatePassword(in.UserInfo.Password)
	if err != nil {
		return nil, rpc.ErrInternal
	}

	userInfo := &db.UserInfo{
		Username:   in.UserInfo.LoginName,
		RealName:   in.UserInfo.RealName,
		PasswordEn: passwordEn,
		IsActive:   in.UserInfo.IsActive,
		IsEditable: in.UserInfo.IsEditable,
		RoleID:     in.UserInfo.RoleId,
	}
	err = db.CreateUser(ctx, userInfo)
	if err != nil {
		log.Warnf("createUser: %v", err)
		if err == db.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		return nil, rpc.ErrInternal
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
				Perms:      convertPerms(roleInfo.PermsJSON),
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

	userInfo, err := db.QueryUserByID(in.UserInfo.Id)
	if err != nil {
		if err == db.ErrRecordNotFound {
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
		userInfo.PasswordEn, err = generatePassword(in.UserInfo.Password)
		if err != nil {
			return nil, rpc.ErrInternal
		}
	}
	var roleInfo *db.UserRole
	if in.UserInfo.RoleId > 0 {
		if roleInfo, err = db.QueryRoleById(ctx, in.UserInfo.RoleId); err != nil {
			log.Warnf("QueryRoleById: %v", err)
			if err == db.ErrRecordNotFound {
				return nil, rpc.ErrNotFound
			}
			return nil, rpc.ErrInternal
		}
		userInfo.RoleID = in.UserInfo.RoleId
	}

	if err = db.UpdateUser(ctx, userInfo); err != nil {
		log.Warnf("updateUser: %v", err)
		if err == db.ErrRecordNotFound {
			return nil, rpc.ErrNotFound
		}
		return nil, rpc.ErrInternal
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
				Perms:      convertPerms(roleInfo.PermsJSON),
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

	users, err := db.QueryUsers(in.UserIds)
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

	err = db.RemoveUsers(users)
	if err != nil {
		log.Infof("RemoveUser user_ids=%v db err=%v", in.UserIds, err)
		return nil, rpc.ErrInternal
	}

	return &pb.RemoveUserReply{}, nil
}

// 用户角色列表
func (s *UserServer) ListRole(ctx context.Context, in *pb.ListRoleRequest) (*pb.ListRoleReply, error) {
	roles, err := db.ListRole()
	if err != nil {
		log.Errorf("ListUser err=%v", err)
		return nil, rpc.ErrInternal
	}

	var reply pb.ListRoleReply
	for _, r := range roles {
		reply.Roles = append(reply.Roles, &pb.UserRole{
			Id:         r.ID,
			Name:       r.Name,
			IsEditable: r.IsEditable,
			Perms:      convertPerms(r.PermsJSON),
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

	role := &db.UserRole{
		Name:       in.RoleInfo.Name,
		IsEditable: in.RoleInfo.IsEditable,
		PermsJSON:  restorePerms(in.RoleInfo.Perms),
	}

	if err := db.CreateRole(ctx, role); err != nil {
		log.Warnf("createUser: %v", err)
		if err == db.ErrDuplicateKey {
			return nil, rpc.ErrAlreadyExists
		}
		return nil, rpc.ErrInternal
	}

	return &pb.CreateRoleReply{
		RoleInfo: &pb.UserRole{
			Id:         role.ID,
			Name:       role.Name,
			IsEditable: role.IsEditable,
			CreatedAt:  role.CreatedAt,
			UpdatedAt:  role.UpdatedAt,
			Perms:      convertPerms(role.PermsJSON),
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

	role, err := db.QueryRoleById(ctx, in.RoleInfo.Id)
	if err != nil && err != db.ErrRecordNotFound {
		log.Warnf("db query role: %v", err)
		return nil, rpc.ErrInternal
	} else if err == db.ErrRecordNotFound {
		return nil, rpc.ErrNotFound
	}

	role.Name = in.RoleInfo.Name
	role.PermsJSON = restorePerms(in.RoleInfo.Perms)

	if err = db.UpdateRole(ctx, role); err != nil {
		log.Warnf("role UpdateRole err: %v", err)
		return nil, err
	}

	return &pb.UpdateRoleReply{
		RoleInfo: &pb.UserRole{
			Id:         role.ID,
			Name:       role.Name,
			IsEditable: role.IsEditable,
			CreatedAt:  role.CreatedAt,
			UpdatedAt:  role.UpdatedAt,
			Perms:      convertPerms(role.PermsJSON),
		},
	}, nil
}

// 删除角色
func (s *UserServer) RemoveRole(ctx context.Context, in *pb.RemoveRoleRequest) (*pb.RemoveRoleReply, error) {
	if in.RoleId <= 0 {
		return nil, rpc.ErrInvalidArgument
	}

	err := db.RemoveRole(ctx, in.RoleId)
	if err != nil {
		log.Warnf("db remove role: %v", err)
		return nil, err
	}
	return &pb.RemoveRoleReply{}, nil
}

func convertPerms(dbPerms []*db.Perm) []*pb.Permission {
	perms := make([]*pb.Permission, 0, len(dbPerms))
	for _, perm := range dbPerms {
		if perm == nil {
			continue
		}
		perms = append(perms, &pb.Permission{
			Id:    perm.ID,
			Allow: perm.Allow,
		})
	}
	return perms
}

func restorePerms(pbPerms []*pb.Permission) []*db.Perm {
	perms := make([]*db.Perm, 0, len(pbPerms))
	for _, perm := range pbPerms {
		if perm == nil {
			continue
		}
		perms = append(perms, &db.Perm{
			ID:    perm.Id,
			Allow: perm.Allow,
		})
	}
	return perms
}
