package utils

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

// 自定义 ReplaceAttr 函数，用于简化源文件路径
// slog.HandlerOptions.ReplaceAttr 是一个函数，用于在日志记录前对每个属性进行处理。
// slog.SourceKey 是源信息属性的键。
// slog.AnyValue(src) 用于将修改后的 *slog.Source 重新封装为 slog.Value。
// filepath.Base(src.File) 将文件路径简化为文件名（如 /home/user/main.go → main.go）。
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey && a.Value.Kind() == slog.KindAny {
		if src, ok := a.Value.Any().(*slog.Source); ok {
			src.File = filepath.Base(src.File) // 只保留文件名
			return slog.Attr{Key: a.Key, Value: slog.AnyValue(src)}
		}
	}
	return a
}

func NewEnhancedLogger(level slog.Leveler, name string, logDir string, when string, interval int) (*slog.Logger, error) {
	if level == nil {
		level = slog.LevelInfo
	}
	// 自动获取脚本名称
	if name == "" {
		name = getScriptName()
	}

	// 确保日志目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// 设置日志文件路径
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", name))

	// 创建带时间滚动的日志文件写入器
	writer, err := rotatelogs.New(
		logFile+".%Y%m%d",
		rotatelogs.WithRotationTime(parseRotationWhen(when, interval)),
	)
	if err != nil {
		return nil, err
	}

	// 创建 slog.Handler（支持结构化日志）
	opts := &slog.HandlerOptions{
		AddSource:   true,
		Level:       level,
		ReplaceAttr: replaceAttr,
	}

	// 同时输出到控制台和文件
	multiWriter := io.MultiWriter(os.Stdout, writer)
	handler := slog.NewTextHandler(multiWriter, opts)

	// 创建统一日志器
	logger := slog.New(handler)

	// 设置为全局日志器（可选）
	slog.SetDefault(logger)

	return logger, nil
}

// 获取当前执行脚本名称
func getScriptName() string {
	if len(os.Args) == 0 {
		return "main"
	}

	scriptPath := os.Args[0]
	base := filepath.Base(scriptPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.ReplaceAll(base, " ", "_")
}

// 将时间单位转换为 time.Duration
func parseRotationWhen(when string, interval int) time.Duration {
	switch when {
	case "midnight", "d", "D":
		return time.Hour * 24
	case "s", "S":
		return time.Second * time.Duration(interval)
	case "m", "M":
		return time.Minute * time.Duration(interval)
	case "h", "H":
		return time.Hour * time.Duration(interval)
	default:
		return time.Hour * 24 // 默认每天滚动
	}
}

var (
	defaultLogger *slog.Logger
	loggerOnce    sync.Once
	initErr       error
)

// InitLogger 初始化全局 logger, 只执行一次
func InitLogger(level slog.Leveler, name string, logDir string, when string, interval int) error {
	loggerOnce.Do(func() {
		logger, err := NewEnhancedLogger(level, name, logDir, when, interval)
		if err != nil {
			initErr = err
			return
		}
		defaultLogger = logger
	})
	return initErr
}

// GetLogger 返回全局 logger 实例
func GetLogger(logDir string) *slog.Logger {
	if defaultLogger == nil {
		if logDir == "" {
			logDir = "logs"
		}

		// 检查 InitLogger 是否返回错误
		if err := InitLogger(nil, "", logDir, "midnight", 1); err != nil {
			panic(fmt.Sprintf("failed to initialize logger: %v", err))
		}
	}
	return defaultLogger
}
