package middlewares

import (
	"context"
	"fmt"
	"gmail-forwarding-system/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// SecurityMiddleware 安全中间件
func SecurityMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 添加安全响应头
		for key, value := range utils.SecurityHeaders {
			c.Header(key, value)
		}

		// 隐藏服务器信息
		c.Header("Server", "Gmail-Forwarding-System")

		c.Next()
	})
}

// RateLimitMiddleware 简单的频率限制中间件
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// 这里可以集成更复杂的限流库，如 golang.org/x/time/rate
	return gin.HandlerFunc(func(c *gin.Context) {
		// 简单实现：可以根据需要集成Redis等存储
		c.Next()
	})
}

// SecurityLoggerMiddleware 安全日志中间件
func SecurityLoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 脱敏IP地址（可选）
		clientIP := param.ClientIP
		if len(clientIP) > 7 {
			clientIP = clientIP[:len(clientIP)-2] + "**"
		}

		return fmt.Sprintf("[%s] %s %s %d %s \"%s\" \"%s\" %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			clientIP,
			param.Method,
			param.StatusCode,
			param.Latency,
			utils.SanitizeInput(param.Path),
			param.ErrorMessage,
			param.Request.UserAgent(),
		)
	})
}

// TimeoutMiddleware 请求超时中间件
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 设置请求超时
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
}
