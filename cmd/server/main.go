// 精弘校园失物招领系统 · 后端服务入口。
//
// 只做四件事：读配置 → 连数据库并迁移 → 装配路由 → 起 HTTP 并优雅退出。
// 依赖装配全部收在 router.New 里，因此这里看不到任何业务细节。
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"lostfound/internal/config"
	"lostfound/internal/database"
	"lostfound/internal/model"
	"lostfound/internal/router"
	"lostfound/pkg/logger"
)

// shutdownTimeout 是收到退出信号后留给在途请求的收尾时间。
const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "服务启动失败:", err)
		os.Exit(1)
	}
}

// run 承载真正的启动流程，把 os.Exit 收敛在 main 一处，便于将来补测试。
func run() error {
	configPath := os.Getenv("APP_CONFIG")
	if configPath == "" {
		configPath = filepath.Join("config", "config.yaml")
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	logger.Init(cfg.Logger.Level)

	db, err := database.Open(cfg.Database)
	if err != nil {
		return err
	}
	if err := database.Migrate(db); err != nil {
		return err
	}
	if cfg.Database.Seed {
		if err := model.Seed(db); err != nil {
			return err
		}
		logger.Info("演示数据已就绪（首次启动写入；库中已有数据则跳过）")
	}

	// 提前建好上传目录：静态文件路由与上传落盘都需要它存在。
	if err := os.MkdirAll(cfg.Upload.Dir, 0o755); err != nil {
		return fmt.Errorf("创建上传目录失败: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router.New(cfg, db),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// HTTP 服务放到后台跑，主协程负责等待退出信号。
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("服务已启动",
			"address", cfg.Address(),
			"mode", cfg.Server.Mode,
			"driver", cfg.Database.Driver,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serveErr:
		return fmt.Errorf("监听失败: %w", err)
	case <-quit:
		logger.Info("收到退出信号，开始优雅关闭")
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("关闭服务时出错", "error", err)
		return err
	}

	logger.Info("服务已退出")
	return nil
}
