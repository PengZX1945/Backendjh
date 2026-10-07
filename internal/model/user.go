// Package model 定义持久化实体与业务状态常量。
//
// 字段的 json tag 直接决定接口返回的字段名，因此这里与接口文档逐字对齐
// （下划线命名、item_status 而非 status……），不做二次映射。
package model

// 用户角色。三种角色对应作业「应用场景」里划分的三类使用者。
const (
	RoleUser        = "user"         // 普通用户
	RoleFinderAdmin = "finder_admin" // 失物招领管理员
	RoleSysAdmin    = "sys_admin"    // 系统管理员
)

// User 是用户账号。
type User struct {
	UserID uint `gorm:"primaryKey;column:user_id" json:"user_id"`
	// Username 登录名，全局唯一。
	Username string `gorm:"size:64;uniqueIndex;not null" json:"username"`
	// Password 存 bcrypt 摘要，永不外泄，因此 json tag 为 "-"。
	Password string `gorm:"size:100;not null" json:"-"`
	Nickname string `gorm:"size:64;not null" json:"nickname"`
	Contact  string `gorm:"size:128;not null" json:"contact"`
	Role     string `gorm:"size:20;not null;default:user;index" json:"role"`
	// Disabled 禁用标记。禁用的账号无法登录，也无法用既有凭证继续调用接口。
	Disabled    bool   `gorm:"not null;default:false" json:"disabled"`
	CreatedTime string `gorm:"size:20;not null" json:"created_time"`
}

// IsBackOffice 判断该用户是否属于后台角色（失物招领管理员及以上）。
// 「及以上」的口径收在这一个方法里，避免各处散写 role 字符串比较而漏掉一种角色。
func (u *User) IsBackOffice() bool {
	return u != nil && (u.Role == RoleFinderAdmin || u.Role == RoleSysAdmin)
}

// IsSysAdmin 判断该用户是否为系统管理员。
func (u *User) IsSysAdmin() bool {
	return u != nil && u.Role == RoleSysAdmin
}
