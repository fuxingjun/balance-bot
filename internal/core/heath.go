package core

import (
	"balance-bot/internal/config"
	"balance-bot/internal/utils"
	"balance-bot/pkg"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

type HealthStatus struct {
	LastHeartbeat int
	NotifyTask    *time.Timer
	WarnCount     int  // 添加警告计数
	IsAlerting    bool // 添加告警状态标识
}

type HealthPayload struct {
	Name  string          `json:"name"`
	Extra json.RawMessage `json:"extra,omitempty"`
}

var (
	healthStore = make(map[string]*HealthStatus)
	storeMutex  sync.RWMutex
)

func HealthCheck(c *fiber.Ctx) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		pkg.GetLogger().Error("Failed to load config", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to load config",
		})
	}
	if cfg == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "config is nil",
		})
	}
	var payload HealthPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid payload",
		})
	}
	pkg.GetLogger().Debug("Health check received", "name", payload.Name, "extra", string(payload.Extra))
	if payload.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "name is required",
		})
	}
	// 记录心跳时间, 秒数
	now := int(time.Now().Unix())
	// 如果不存在则初始化
	storeMutex.Lock()
	defer storeMutex.Unlock()

	status, exists := healthStore[payload.Name]
	if !exists {
		// 初始化新的健康状态
		status = &HealthStatus{
			LastHeartbeat: now,
			WarnCount:     0,
			IsAlerting:    false,
		}
		healthStore[payload.Name] = status
	} else {
		// 停止现有的告警任务
		if status.NotifyTask != nil {
			status.NotifyTask.Stop()
			status.NotifyTask = nil
		}

		// 重置告警状态
		status.LastHeartbeat = now
		status.WarnCount = 0
		status.IsAlerting = false
	}

	// 设置新的告警任务
	status.NotifyTask = time.AfterFunc(time.Duration(cfg.HealthCheck.Interval)*time.Second, func() {
		handleHealthCheckTimeout(payload.Name, status, cfg)
	})

	return c.JSON(fiber.Map{
		"status": "ok",
		"data": fiber.Map{
			"lastHeartbeat": formatTime(status.LastHeartbeat),
			"nextCheck":     formatTime(status.LastHeartbeat + cfg.HealthCheck.Interval),
		},
	})
}

// 处理超时告警的独立函数
func handleHealthCheckTimeout(name string, status *HealthStatus, cfg *config.AppConfig) {
	storeMutex.Lock()
	if status.IsAlerting {
		storeMutex.Unlock()
		return
	}
	status.IsAlerting = true
	storeMutex.Unlock()

	for i := 0; i < cfg.HealthCheck.WarnCount; i++ {
		storeMutex.RLock()
		lastBeat := status.LastHeartbeat
		storeMutex.RUnlock()
		msg := fmt.Sprintf("⚠️ Health check timeout for %s, last heartbeat at %s", name, formatTime(lastBeat))

		pkg.GetLogger().Warn(msg)

		if err := utils.SendMessage(msg); err != nil {
			pkg.GetLogger().Error("Failed to send alert", "error", err, "service", name, "attempt", i+1)
		}

		time.Sleep(time.Duration(cfg.HealthCheck.Interval) * time.Second)

		// 检查是否已经收到新的心跳
		storeMutex.RLock()
		if status.LastHeartbeat > lastBeat {
			storeMutex.RUnlock()
			break
		}
		storeMutex.RUnlock()
	}

	storeMutex.Lock()
	status.IsAlerting = false
	storeMutex.Unlock()
}

// 格式化时间带时区
func formatTime(t int) string {
	return time.Unix(int64(t), 0).Format(time.RFC3339)
}
