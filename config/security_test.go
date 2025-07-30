package config

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	config := DefaultSecurityConfig
	limiter := &RateLimiter{}

	t.Run("InitialState", func(t *testing.T) {
		// 初始状态应该允许发送
		if !limiter.CanSend(config) {
			t.Error("Initial state should allow sending")
		}
	})

	t.Run("MinIntervalLimit", func(t *testing.T) {
		// 记录一次发送
		limiter.RecordSend()

		// 立即检查，应该被限制
		if limiter.CanSend(config) {
			t.Error("Should be limited by minimum interval")
		}

		// 等待足够的时间（模拟）
		limiter.LastSendTime = time.Now().Add(-config.MinSendInterval - time.Second)

		if !limiter.CanSend(config) {
			t.Error("Should be allowed after minimum interval")
		}
	})

	t.Run("HourlyLimit", func(t *testing.T) {
		limiter = &RateLimiter{
			LastSendTime:    time.Now().Add(-time.Hour),
			HourlySendCount: config.MaxHourlySends,
			CurrentHour:     time.Now().Hour(),
		}

		if limiter.CanSend(config) {
			t.Error("Should be limited by hourly count")
		}

		// 重置小时计数
		limiter.CurrentHour = time.Now().Hour() - 1

		if !limiter.CanSend(config) {
			t.Error("Should be allowed after hour reset")
		}
	})

	t.Run("DailyLimit", func(t *testing.T) {
		limiter = &RateLimiter{
			LastSendTime:   time.Now().Add(-time.Hour),
			DailySendCount: config.DailySendLimit,
			CurrentDay:     time.Now().YearDay(),
		}

		if limiter.CanSend(config) {
			t.Error("Should be limited by daily count")
		}

		// 重置日计数
		limiter.CurrentDay = time.Now().YearDay() - 1

		if !limiter.CanSend(config) {
			t.Error("Should be allowed after day reset")
		}
	})

	t.Run("RecordSend", func(t *testing.T) {
		limiter = &RateLimiter{}
		initialHourly := limiter.HourlySendCount
		initialDaily := limiter.DailySendCount

		limiter.RecordSend()

		if limiter.HourlySendCount != initialHourly+1 {
			t.Errorf("Expected hourly count %d, got %d", initialHourly+1, limiter.HourlySendCount)
		}

		if limiter.DailySendCount != initialDaily+1 {
			t.Errorf("Expected daily count %d, got %d", initialDaily+1, limiter.DailySendCount)
		}

		if limiter.LastSendTime.IsZero() {
			t.Error("LastSendTime should be set")
		}
	})
}

func TestDefaultSecurityConfig(t *testing.T) {
	config := DefaultSecurityConfig

	t.Run("DefaultValues", func(t *testing.T) {
		if !config.EnableHTTPS {
			t.Error("HTTPS should be enabled by default")
		}

		if config.APIRateLimit <= 0 {
			t.Error("API rate limit should be positive")
		}

		if config.MaxRetryTimes <= 0 {
			t.Error("Max retry times should be positive")
		}

		if config.TimeoutSeconds <= 0 {
			t.Error("Timeout seconds should be positive")
		}

		if config.MinSendInterval <= 0 {
			t.Error("Min send interval should be positive")
		}

		if config.MaxHourlySends <= 0 {
			t.Error("Max hourly sends should be positive")
		}

		if config.DailySendLimit <= 0 {
			t.Error("Daily send limit should be positive")
		}
	})

	t.Run("ReasonableDefaults", func(t *testing.T) {
		// 检查默认值是否合理
		if config.APIRateLimit > 1000 {
			t.Error("API rate limit seems too high")
		}

		if config.MaxRetryTimes > 10 {
			t.Error("Max retry times seems too high")
		}

		if config.MinSendInterval > time.Minute {
			t.Error("Min send interval seems too long")
		}

		if config.MaxHourlySends > 1000 {
			t.Error("Max hourly sends seems too high")
		}
	})
}
