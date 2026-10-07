// Package token 负责签发与校验登录凭证（JWT，HS256）。
//
// 选 JWT 而不是服务端 Session：作业要求「后端使用常见的鉴权方式」且前端要做
// 「登录状态持久化」。JWT 自包含、无状态、天然适配前后端分离，前端把 token 存进
// localStorage 后按 Authorization: Bearer 携带即可，后端无需维护会话存储。
package token

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"lostfound/internal/model"
)

// Claims 是写入凭证的载荷。
//
// 刻意只放 user_id 与 username，不放 role：角色以数据库为准，这样管理员刚调整完
// 某人的角色，其下一个请求立刻生效，不必等旧 token 过期。username 仅用于日志排查。
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// Manager 签发与解析凭证。
type Manager struct {
	secret []byte
	expire time.Duration
	issuer string
}

// NewManager 构造凭证管理器。expireHours 为有效期小时数。
func NewManager(secret string, expireHours int, issuer string) *Manager {
	return &Manager{
		secret: []byte(secret),
		expire: time.Duration(expireHours) * time.Hour,
		issuer: issuer,
	}
}

// Issue 为指定用户签发凭证。
func (m *Manager) Issue(user *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   user.UserID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   strconv.FormatUint(uint64(user.UserID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Parse 校验凭证并返回其中的用户 ID。
// 签名不符、算法被篡改、已过期都会返回错误，由调用方统一按「未登录」处理。
func (m *Manager) Parse(rawToken string) (uint, error) {
	var claims Claims
	parsed, err := jwt.ParseWithClaims(rawToken, &claims, func(parsed *jwt.Token) (any, error) {
		// 固定校验签名算法，挡掉 alg=none 与「HMAC 换成非对称算法」这类篡改。
		if _, ok := parsed.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", parsed.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return 0, fmt.Errorf("解析凭证失败: %w", err)
	}
	if !parsed.Valid {
		return 0, errors.New("凭证无效")
	}
	if claims.UserID == 0 {
		return 0, errors.New("凭证缺少用户标识")
	}
	return claims.UserID, nil
}
