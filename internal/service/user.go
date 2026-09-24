package service

import (
	"Backendjh/internal/model"
	"Backendjh/internal/pkg/errcode"
	"Backendjh/internal/pkg/jwt"
	"Backendjh/internal/repository"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{4,32}$`)

type UserService struct{ JWTSecret string }

func (s *UserService) Register(username, password, nickname, contact string) (*model.User, *errcode.Error) {
	if !usernameRe.MatchString(username) || len(password) < 8 || len(password) > 64 {
		return nil, errcode.ParamError
	}
	if nickname == "" || contact == "" || len(nickname) > 32 {
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
	token, err := jwt.Generate(s.JWTSecret, u.ID, u.Role)
	if err != nil {
		return "", nil, errcode.InternalError
	}
	return token, u, nil
}
