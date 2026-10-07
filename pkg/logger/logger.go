// Package logger 是全局日志出口，包一层标准库 log/slog。
//
// 不引第三方日志库：slog 已是标准库，结构化、分级、可换 Handler，够用且零依赖。
// 全站只通过本包打日志，将来换实现（加文件轮转、接采集）只需改这里。
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// base 是全局 logger。Init 之前也有一份可用的默认值，避免空指针。
var base = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

// Init 按配置的级别重建全局 logger，并同步给标准库默认 logger。
func Init(level string) {
	base = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)}))
	slog.SetDefault(base)
}

// parseLevel 把配置里的级别名转成 slog 级别，无法识别时回落为 info。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug 打调试日志。
func Debug(msg string, args ...any) { base.Debug(msg, args...) }

// Info 打常规日志。
func Info(msg string, args ...any) { base.Info(msg, args...) }

// Warn 打警告日志。
func Warn(msg string, args ...any) { base.Warn(msg, args...) }

// Error 打错误日志。
func Error(msg string, args ...any) { base.Error(msg, args...) }
