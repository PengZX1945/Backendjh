package repository

import (
	"gorm.io/gorm"

	"lostfound/internal/model"
)

// UserRepository 提供用户的数据访问。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 构造用户仓储。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 新增用户。
func (r *UserRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// FindByID 按主键查询，不存在时返回 ErrNotFound。
func (r *UserRepository) FindByID(userID uint) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, "user_id = ?", userID).Error; err != nil {
		return nil, translate(err)
	}
	return &user, nil
}

// FindByUsername 按登录名查询，不存在时返回 ErrNotFound。
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.First(&user, "username = ?", username).Error; err != nil {
		return nil, translate(err)
	}
	return &user, nil
}

// ExistsByUsername 判断登录名是否已被占用，用于注册查重。
func (r *UserRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// Update 保存用户的可变字段（昵称、联系方式、角色、密码、禁用标记）。
func (r *UserRepository) Update(user *model.User) error {
	return r.db.Model(&model.User{}).
		Where("user_id = ?", user.UserID).
		Updates(map[string]any{
			"nickname": user.Nickname,
			"contact":  user.Contact,
			"role":     user.Role,
			"password": user.Password,
			"disabled": user.Disabled,
		}).Error
}

// List 按角色与关键字分页查询用户列表，关键字匹配用户名或昵称。
func (r *UserRepository) List(role, keyword string, offset, limit int) ([]model.User, error) {
	query := r.db.Model(&model.User{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	var users []model.User
	if err := query.Order("user_id asc").Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Count 统计用户总数，供演示数据初始化时的幂等判断使用。
func (r *UserRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
