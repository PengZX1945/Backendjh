package middleware

import (
	"errors"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"

	"lostfound/internal/model"
	"lostfound/internal/repository"
	"lostfound/internal/token"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
)

// contextUserKey 是当前登录用户在 gin.Context 中的键。
// 用带包名前缀的私有常量，避免与其他中间件写裸字符串撞键。
const contextUserKey = "lostfound.currentUser"

// bearerPrefix 是 Authorization 头里凭证的前缀。
const bearerPrefix = "Bearer "

// CurrentUser 取出当前登录用户。未登录、或可选鉴权未命中时返回 nil。
func CurrentUser(c *gin.Context) *model.User {
	value, exists := c.Get(contextUserKey)
	if !exists {
		return nil
	}
	user, ok := value.(*model.User)
	if !ok {
		return nil
	}
	return user
}

// Authenticate 要求必须登录，未通过则回 401 + 错误码 2。
func Authenticate(tokens *token.Manager, users *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := resolveUser(c, tokens, users)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		c.Set(contextUserKey, user)
		c.Next()
	}
}

// OptionalAuthenticate 解析登录态但不强制。
//
// 用于公开接口（如物品详情）需要按访问者身份调整可见范围的场景：凭证缺失或非法
// 都按游客继续往下走，不打断请求 —— 拿一个过期 token 也应当能浏览公开内容。
func OptionalAuthenticate(tokens *token.Manager, users *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		if user, err := resolveUser(c, tokens, users); err == nil {
			c.Set(contextUserKey, user)
		}
		c.Next()
	}
}

// RequireRole 要求当前用户具备给定角色之一，否则回 403 + 错误码 3。
//
// 它自己完成「验凭证」与「查角色」两步，因此使用时不必再串一道 Authenticate。
// 这样设计是有教训的：若让角色判断只依赖上下文里已有的用户，一旦漏配鉴权中间件，
// 所有后台接口都会因为「用户为空」而统一返回未登录 —— 一个既不像权限错误、
// 又很难从返回码上看出来的失效方式。把两步合成一个中间件，误用的可能性就没有了。
func RequireRole(
	tokens *token.Manager,
	users *repository.UserRepository,
	roles ...string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := resolveUser(c, tokens, users)
		if err != nil {
			c.Error(err)
			c.Abort()
			return
		}
		if !slices.Contains(roles, user.Role) {
			c.Error(apperr.New(errcode.Forbidden))
			c.Abort()
			return
		}
		c.Set(contextUserKey, user)
		c.Next()
	}
}

// resolveUser 解析 Authorization 头里的凭证并加载用户。
//
// 角色与联系方式每次都从数据库读，而不是取 token 载荷里的旧值：管理员刚调整完某人的
// 角色，其下一个请求立刻生效，不必等旧凭证过期。这也是载荷里不放 role 的原因。
func resolveUser(
	c *gin.Context,
	tokens *token.Manager,
	users *repository.UserRepository,
) (*model.User, error) {
	rawToken, err := bearerToken(c)
	if err != nil {
		return nil, apperr.New(errcode.Unauthorized)
	}

	userID, err := tokens.Parse(rawToken)
	if err != nil {
		return nil, apperr.Wrap(errcode.Unauthorized, err)
	}

	user, err := users.FindByID(userID)
	if err != nil {
		// 凭证本身有效但用户已不存在（被删除）：同样按未登录处理，让前端跳登录页。
		return nil, apperr.New(errcode.Unauthorized)
	}
	if user.Disabled {
		// 文档为「被禁用的账号」单列了错误码 10。除了拦住登录，这里也拦住既有凭证 ——
		// 否则封禁一个账号要等它的凭证过期才生效。
		return nil, apperr.New(errcode.AccountDisabled)
	}
	return user, nil
}

// bearerToken 从 Authorization 头里取出 Bearer 凭证本体。
func bearerToken(c *gin.Context) (string, error) {
	header := strings.TrimSpace(c.GetHeader("Authorization"))
	if header == "" {
		return "", errors.New("缺少 Authorization 头")
	}
	if len(header) <= len(bearerPrefix) || !strings.EqualFold(header[:len(bearerPrefix)], bearerPrefix) {
		return "", errors.New("Authorization 头格式不正确")
	}
	return strings.TrimSpace(header[len(bearerPrefix):]), nil
}
