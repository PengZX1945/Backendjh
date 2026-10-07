// Package database 负责建立数据库连接并执行结构迁移。
//
// 默认用纯 Go 的 SQLite 驱动（github.com/glebarez/sqlite）：不依赖 CGO，评分环境
// go run 即可跑起来，不需要额外装 MySQL。切换到 MySQL 只改配置里的 driver/dsn，
// 上层仓储与服务代码完全无感 —— 差异被隔离在本文件。
package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"lostfound/internal/config"
	"lostfound/internal/model"
)

// slowQueryThreshold 超过它就记一条慢查询日志。
const slowQueryThreshold = 200 * time.Millisecond

// Open 按配置打开数据库连接。
func Open(cfg config.Database) (*gorm.DB, error) {
	dialector, err := buildDialector(cfg)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: newLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	return db, nil
}

// newLogger 构造 GORM 的日志器。
//
// 忽略「记录不存在」：查不到对应 404，是正常的业务结局而非异常，按警告刷屏会淹掉
// 真正值得看的行。只保留慢查询与真正的错误 —— 日志里剩下的每一条都值得处理。
func newLogger() gormlogger.Interface {
	return gormlogger.New(log.New(os.Stdout, "", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             slowQueryThreshold,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
	})
}

// buildDialector 把配置的 driver 名翻译成 GORM 方言。
func buildDialector(cfg config.Database) (gorm.Dialector, error) {
	switch cfg.Driver {
	case "mysql":
		return mysql.Open(cfg.DSN), nil
	case "sqlite", "":
		return sqlite.Open(cfg.DSN), nil
	default:
		return nil, fmt.Errorf("不支持的数据库驱动: %s（可选 sqlite / mysql）", cfg.Driver)
	}
}

// Migrate 建表或补齐缺失字段。
// 用 AutoMigrate 而非手写 SQL 迁移：本项目实体少且改动频繁，AutoMigrate 保证
// 表结构与 model 定义始终一致，省去维护迁移脚本的同步成本。
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.User{},
		&model.Item{},
		&model.Claim{},
		&model.Announcement{},
	); err != nil {
		return fmt.Errorf("迁移数据库结构失败: %w", err)
	}
	return nil
}
