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
