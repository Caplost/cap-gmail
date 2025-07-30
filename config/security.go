package config

import (
	"time"
)

// SecurityConfig 安全配置
type SecurityConfig struct {
	// API 安全配置
	EnableHTTPS    bool     `json:"enable_https"`
	APIRateLimit   int      `json:"api_rate_limit"`  // 每分钟API调用限制
	MaxRetryTimes  int      `json:"max_retry_times"` // 最大重试次数
	TimeoutSeconds int      `json:"timeout_seconds"` // 请求超时时间
	AllowedOrigins []string `json:"allowed_origins"` // 允许的跨域来源

	// 邮件发送安全配置
	MinSendInterval time.Duration `json:"min_send_interval"` // 最小发送间隔
	MaxHourlySends  int           `json:"max_hourly_sends"`  // 每小时最大发送数量
	DailySendLimit  int           `json:"daily_send_limit"`  // 每日发送限制

	// 数据安全配置
	LogRetentionDays int           `json:"log_retention_days"` // 日志保留天数
	EnableDataMask   bool          `json:"enable_data_mask"`   // 启用数据脱敏
	BackupInterval   time.Duration `json:"backup_interval"`    // 备份间隔

	// 监控告警配置
	MaxFailureRate float64  `json:"max_failure_rate"` // 最大失败率阈值
	AlertEmailList []string `json:"alert_email_list"` // 告警邮件列表
	HealthCheckURL string   `json:"health_check_url"` // 健康检查URL
}

// DefaultSecurityConfig 默认安全配置
var DefaultSecurityConfig = SecurityConfig{
	// API 安全
	EnableHTTPS:    true,
	APIRateLimit:   60, // 每分钟60次
	MaxRetryTimes:  3,
	TimeoutSeconds: 30,
	AllowedOrigins: []string{"*"},

	// 邮件发送安全
	MinSendInterval: 30 * time.Second, // 最小间隔30秒
	MaxHourlySends:  100,              // 每小时100封
	DailySendLimit:  1000,             // 每日1000封

	// 数据安全
	LogRetentionDays: 30,             // 保留30天日志
	EnableDataMask:   true,           // 启用数据脱敏
	BackupInterval:   24 * time.Hour, // 每24小时备份一次

	// 监控告警
	MaxFailureRate: 0.05, // 5%失败率告警
	AlertEmailList: []string{},
	HealthCheckURL: "/health",
}

// MonitoringMetrics 监控指标
type MonitoringMetrics struct {
	FailedForwards  int       `json:"failed_forwards"`   // 转发失败次数
	APIErrorRate    float64   `json:"api_error_rate"`    // API错误率
	ProcessingDelay int64     `json:"processing_delay"`  // 处理延迟(毫秒)
	DailyVolume     int       `json:"daily_volume"`      // 日处理量
	ActiveTargets   int       `json:"active_targets"`    // 活跃转发对象数
	LastProcessTime time.Time `json:"last_process_time"` // 最后处理时间
	SystemUptime    int64     `json:"system_uptime"`     // 系统运行时间(秒)
}

// RateLimiter 频率限制器结构
type RateLimiter struct {
	LastSendTime    time.Time `json:"last_send_time"`
	HourlySendCount int       `json:"hourly_send_count"`
	DailySendCount  int       `json:"daily_send_count"`
	CurrentHour     int       `json:"current_hour"`
	CurrentDay      int       `json:"current_day"`
}

// CanSend 检查是否可以发送
func (rl *RateLimiter) CanSend(config SecurityConfig) bool {
	now := time.Now()

	// 检查最小间隔
	if now.Sub(rl.LastSendTime) < config.MinSendInterval {
		return false
	}

	// 重置计数器（小时）
	if now.Hour() != rl.CurrentHour {
		rl.HourlySendCount = 0
		rl.CurrentHour = now.Hour()
	}

	// 重置计数器（天）
	if now.YearDay() != rl.CurrentDay {
		rl.DailySendCount = 0
		rl.CurrentDay = now.YearDay()
	}

	// 检查小时限制
	if rl.HourlySendCount >= config.MaxHourlySends {
		return false
	}

	// 检查日限制
	if rl.DailySendCount >= config.DailySendLimit {
		return false
	}

	return true
}

// RecordSend 记录发送
func (rl *RateLimiter) RecordSend() {
	rl.LastSendTime = time.Now()
	rl.HourlySendCount++
	rl.DailySendCount++
}
