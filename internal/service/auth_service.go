package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"lostfound/internal/dto"
	"lostfound/internal/model"
	"lostfound/internal/repository"
	"lostfound/internal/token"
	"lostfound/pkg/apperr"
	"lostfound/pkg/errcode"
	"lostfound/pkg/timeutil"
)

// defaultUserPageSize 是用户列表的默认分页大小。
// 与信息流的 12 不同：用户管理是表格视图，一屏能放更多行。
const defaultUserPageSize = 20

// AuthService 处理注册、登录、个人资料与用户管理。
type AuthService struct {
	users  *repository.UserRepository
	tokens *token.Manager
}

// NewAuthService 构造鉴权服务。
func NewAuthService(users *repository.UserRepository, tokens *token.Manager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// Register 注册普通用户。
//
// 角色固定为 user：注册接口不接受角色入参，管理员只能由系统管理员事后调整，
// 否则任何人都能自己注册成系统管理员。
func (s *AuthService) Register(req dto.RegisterRequest) error {
	if err := requireText(req.Username, req.Password, req.Nickname, req.Contact); err != nil {
		return err
	}

	taken, err := s.users.ExistsByUsername(req.Username)
	if err != nil {
		return err
	}
	if taken {
		return apperr.New(errcode.UsernameTaken)
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.users.Create(&model.User{
		Username:    req.Username,
		Password:    string(digest),
		Nickname:    req.Nickname,
		Contact:     req.Contact,
		Role:        model.RoleUser,
		CreatedTime: timeutil.Now(),
	})
	if err != nil {
		// 并发注册同名账号时，上面的查重会双双通过，最终由唯一索引拦下。
		// 这里把这类冲突还原成语义正确的 5，而不是让它冒泡成 114。
		if stillTaken, checkErr := s.users.ExistsByUsername(req.Username); checkErr == nil && stillTaken {
			return apperr.New(errcode.UsernameTaken)
		}
		return err
	}
	return nil
}

// Login 校验凭证并签发 token。
//
// 判定顺序与接口文档一致：先用户名/密码(6)，再账号禁用(10)。密码错时不区分
// 「账号不存在」与「密码不对」，也不提前暴露「该账号已禁用」。
func (s *AuthService) Login(req dto.LoginRequest) (dto.LoginResponse, error) {
	if err := requireText(req.Username, req.Password); err != nil {
		return dto.LoginResponse{}, err
	}

	user, err := s.users.FindByUsername(req.Username)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.LoginResponse{}, apperr.New(errcode.BadCredentials)
	}
	if err != nil {
		return dto.LoginResponse{}, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) != nil {
		return dto.LoginResponse{}, apperr.New(errcode.BadCredentials)
	}
	if user.Disabled {
		return dto.LoginResponse{}, apperr.New(errcode.AccountDisabled)
	}

	signed, err := s.tokens.Issue(user)
	if err != nil {
		return dto.LoginResponse{}, err
	}
	return dto.LoginResponse{Token: signed}, nil
}

// Logout 退出登录。
//
// JWT 是无状态的，服务端没有会话可销毁：客户端丢弃凭证即完成退出。保留该接口是为
// 对齐文档，也让前端有一个语义明确的注销动作可调用（便于将来换成服务端黑名单）。
func (s *AuthService) Logout() error {
	return nil
}

// Profile 读取当前登录用户的档案。
func (s *AuthService) Profile(userID uint) (dto.ProfileResponse, error) {
	user, err := s.users.FindByID(userID)
	if err != nil {
		return dto.ProfileResponse{}, translate(err)
	}
	return dto.ProfileResponse{
		ID:       user.UserID,
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
		Contact:  user.Contact,
	}, nil
}

// UpdateProfile 修改昵称与联系方式。
//
// 文档把 user_id 列为 query 参数（「从 auth 传入」）。该值以鉴权中间件注入的身份为准，
// 若调用方另传了一个不属于自己的 user_id，判越权(3)。
func (s *AuthService) UpdateProfile(userID, queryUserID uint, req dto.UpdateProfileRequest) error {
	if queryUserID != 0 && queryUserID != userID {
		return apperr.New(errcode.Forbidden)
	}
	if err := requireText(req.Nickname, req.Contact); err != nil {
		return err
	}

	user, err := s.users.FindByID(userID)
	if err != nil {
		return translate(err)
	}
	user.Nickname = req.Nickname
	user.Contact = req.Contact
	return s.users.Update(user)
}

// ChangePassword 修改密码。
// 旧密码不符判参数错误(1)；新旧密码相同判 11（文档为本条单列了错误码）。
func (s *AuthService) ChangePassword(userID, queryUserID uint, req dto.ChangePasswordRequest) error {
	if queryUserID != 0 && queryUserID != userID {
		return apperr.New(errcode.Forbidden)
	}
	if err := requireText(req.OldPassword, req.NewPassword); err != nil {
		return err
	}

	user, err := s.users.FindByID(userID)
	if err != nil {
		return translate(err)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)) != nil {
		return apperr.New(errcode.BadRequest)
	}
	if req.NewPassword == req.OldPassword {
		return apperr.New(errcode.SamePassword)
	}

	digest, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(digest)
	return s.users.Update(user)
}

// ListUsers 用户列表，仅供系统管理员。
func (s *AuthService) ListUsers(query dto.UserQuery) (dto.UserListResponse, error) {
	if query.PageSize <= 0 {
		query.PageSize = defaultUserPageSize
	}

	users, err := s.users.List(query.Role, query.Keyword, query.Offset(), query.PageSize)
	if err != nil {
		return dto.UserListResponse{}, err
	}

	records := make([]dto.ManagedUserResponse, 0, len(users))
	for index := range users {
		user := &users[index]
		records = append(records, dto.ManagedUserResponse{
			UserID:   user.UserID,
			Username: user.Username,
			Nickname: user.Nickname,
			Role:     user.Role,
			Contact:  user.Contact,
		})
	}
	return dto.UserListResponse{Users: records}, nil
}

// UpdateUserRole 调整用户角色：设为 finder_admin 即新增管理员，改回 user 即移除。
//
// 分支顺序是刻意的：先判「改自己」→ 再判「目标是系统管理员」→ 最后校验角色取值。
// 若把「目标是系统管理员」提前，系统管理员降级自己时会先撞上「不允许修改其他系统
// 管理员」，返回 3 而不是文档要求的 7 —— 两条规则同时成立时，「自己」更具体，应当优先。
func (s *AuthService) UpdateUserRole(operatorID, targetUserID uint, role string) error {
	if targetUserID == operatorID {
		return apperr.New(errcode.InvalidState)
	}

	target, err := s.users.FindByID(targetUserID)
	if err != nil {
		return translate(err)
	}
	if target.Role == model.RoleSysAdmin {
		return apperr.New(errcode.Forbidden)
	}
	if role != model.RoleFinderAdmin && role != model.RoleUser {
		return apperr.New(errcode.BadRequest)
	}

	target.Role = role
	return s.users.Update(target)
}
