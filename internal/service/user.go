package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/jwt"
	"Backendjh/internal/repository"
	"log"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{4,32}$`)

type UserService struct{ jwtSecret string }

func NewUserService(jwtSecret string) *UserService {
	return &UserService{jwtSecret: jwtSecret}
}

func (s *UserService) Register(username, password, nickname, contact string) (*model.User, *errcode.Error) {
	if !usernameRe.MatchString(username) || len(password) < 8 || len(password) > 64 {
		return nil, errcode.ParamError
	}
	if nickname == "" || contact == "" || len(nickname) > 32 || len(contact) > 64 {
		return nil, errcode.ParamError
	}
	exists, err := repository.FindUserByUsername(username)
	if err != nil {
		return nil, errcode.InternalError
	}
	if exists != nil {
		return nil, errcode.UsernameExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errcode.InternalError
	}
	u := &model.User{Username: username, PasswordHash: string(hash), Nickname: nickname, Contact: contact, Role: model.RoleUser}

	if err := repository.CreateUser(u); err != nil {
		return nil, errcode.InternalError
	}
	return u, nil
}

func (s *UserService) Login(username, password string) (string, *model.User, *errcode.Error) {
	u, err := repository.FindUserByUsername(username)
	if err != nil {
		return "", nil, errcode.InternalError
	}
	if u == nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", nil, errcode.WrongPassword
	}
	if u.Status != 1 {
		return "", nil, errcode.AccountDisabled
	}
	token, err := jwt.Generate(s.jwtSecret, u.ID, u.Role)
	if err != nil {
		return "", nil, errcode.InternalError
	}
	return token, u, nil
}

func (s *UserService) UpdateProfile(userID uint64, nickname, contact string) (*model.User, *errcode.Error) {
	u, err := repository.FindUserByID(userID)
	if err != nil {
		return nil, errcode.InternalError
	}
	if u == nil {
		return nil, errcode.NotFound
	}
	if nickname == "" || contact == "" || len(nickname) > 32 || len(contact) > 64 {
		return nil, errcode.ParamError
	}
	u.Nickname, u.Contact = nickname, contact
	err = repository.UpdateUser(u)
	if err != nil {
		return nil, errcode.InternalError
	}
	return u, nil
}

func (s *UserService) ChangePassword(userID uint64, oldPassword, newPassword string) *errcode.Error {
	if len(newPassword) < 8 || len(newPassword) > 64 {
		return errcode.ParamError
	}
	if oldPassword == newPassword {
		return errcode.SamePassword
	}
	u, err := repository.FindUserByID(userID)
	if err != nil {
		return errcode.InternalError
	}
	if u == nil {
		return errcode.NotFound
	}
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPassword))
	if err != nil {
		return errcode.WrongPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

	if err != nil {
		return errcode.InternalError
	}
	u.PasswordHash = string(hash)
	err = repository.UpdateUser(u)
	if err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *UserService) GetProfile(userID uint64) (*model.User, *errcode.Error) {
	u, err := repository.FindUserByID(userID)
	if err != nil {
		return nil, errcode.InternalError
	}
	if u == nil {
		return nil, errcode.NotFound
	}
	return u, nil
}

func (s *UserService) AdminUserList(role, keyword, roleFilter string, page, pageSize int) ([]model.User, *errcode.Error) {
	if role != model.RoleSysAdmin {
		return nil, errcode.Forbidden
	}
	users, err := repository.ListUsers(repository.UserListFilter{
		Keyword: keyword, Role: roleFilter, Page: page, PageSize: pageSize,
	})
	if err != nil {
		return nil, errcode.InternalError
	}
	return users, nil
}

func (s *UserService) UpdateRole(operatorID uint64, operatorRole string, targetID uint64, newRole string) *errcode.Error {
	if operatorRole != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	if newRole != model.RoleUser && newRole != model.RoleFinderAdmin {
		return errcode.ParamError
	}
	if targetID == operatorID {
		return errcode.StatusNotAllowed
	}
	target, err := repository.FindUserByID(targetID)
	if err != nil {
		return errcode.InternalError
	}
	if target == nil {
		return errcode.NotFound
	}
	if target.Role == model.RoleSysAdmin {
		return errcode.Forbidden
	}
	target.Role = newRole
	if err := repository.UpdateUser(target); err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *UserService) UpdateStatus(operatorID uint64, operatorRole string, targetID uint64, status int8) *errcode.Error {
	if operatorRole != model.RoleSysAdmin {
		return errcode.Forbidden
	}
	if status != model.UserStatusDisabled && status != model.UserStatusEnabled {
		return errcode.ParamError
	}
	if targetID == operatorID {
		return errcode.StatusNotAllowed
	}
	target, err := repository.FindUserByID(targetID)
	if err != nil {
		return errcode.InternalError
	}
	if target == nil {
		return errcode.NotFound
	}
	if target.Role == model.RoleSysAdmin {
		return errcode.Forbidden
	}
	target.Status = status
	if err := repository.UpdateUser(target); err != nil {
		return errcode.InternalError
	}
	return nil
}

func (s *UserService) SeedSysAdmin(username, password string) {
	if password == "" {
		return
	}
	var n int64
	if err := model.DB.Model(&model.User{}).
		Where("role = ?", model.RoleSysAdmin).Count(&n).Error; err != nil || n > 0 {
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	u := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     "系统管理员",
		Role:         model.RoleSysAdmin,
		Status:       1,
	}
	if err := repository.CreateUser(u); err == nil {
		log.Printf("seeded sys_admin: %s", username)
	}
}
