package utils

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// 保存原始的输出
	originalOutput := log.Default().Writer()
	defer log.SetOutput(originalOutput)

	// 创建缓冲区来捕获日志输出
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	t.Run("InitLogger", func(t *testing.T) {
		InitLogger()

		if Logger == nil {
			t.Error("Expected Logger to be initialized")
		}
	})

	t.Run("LogInfo", func(t *testing.T) {
		logBuffer.Reset()

		// 重定向Logger输出到缓冲区
		Logger = log.New(&logBuffer, "[Gmail-Forwarding] ", log.LstdFlags|log.Lshortfile)

		LogInfo("测试信息日志")

		output := logBuffer.String()
		if !strings.Contains(output, "[INFO]") {
			t.Error("Expected [INFO] prefix in log output")
		}
		if !strings.Contains(output, "测试信息日志") {
			t.Error("Expected log message in output")
		}
		if !strings.Contains(output, "[Gmail-Forwarding]") {
			t.Error("Expected logger prefix in output")
		}
	})

	t.Run("LogError", func(t *testing.T) {
		logBuffer.Reset()

		Logger = log.New(&logBuffer, "[Gmail-Forwarding] ", log.LstdFlags|log.Lshortfile)

		// 测试带错误的日志
		testErr := fmt.Errorf("测试错误")
		LogError("发生错误", testErr)

		output := logBuffer.String()
		if !strings.Contains(output, "[ERROR]") {
			t.Error("Expected [ERROR] prefix in log output")
		}
		if !strings.Contains(output, "发生错误") {
			t.Error("Expected error message in output")
		}
		if !strings.Contains(output, "测试错误") {
			t.Error("Expected error details in output")
		}

		// 测试不带错误的日志
		logBuffer.Reset()
		LogError("错误消息", nil)

		output = logBuffer.String()
		if !strings.Contains(output, "错误消息") {
			t.Error("Expected error message without error details")
		}
	})

	t.Run("LogWarn", func(t *testing.T) {
		logBuffer.Reset()

		Logger = log.New(&logBuffer, "[Gmail-Forwarding] ", log.LstdFlags|log.Lshortfile)

		LogWarn("测试警告日志")

		output := logBuffer.String()
		if !strings.Contains(output, "[WARN]") {
			t.Error("Expected [WARN] prefix in log output")
		}
		if !strings.Contains(output, "测试警告日志") {
			t.Error("Expected warning message in output")
		}
	})

	t.Run("LoggerFormat", func(t *testing.T) {
		logBuffer.Reset()

		Logger = log.New(&logBuffer, "[Test-Prefix] ", log.LstdFlags)

		LogInfo("格式测试")

		output := logBuffer.String()

		// 检查是否包含时间戳
		if !strings.Contains(output, "/") && !strings.Contains(output, ":") {
			t.Error("Expected timestamp in log output")
		}

		// 检查前缀
		if !strings.Contains(output, "[Test-Prefix]") {
			t.Error("Expected custom prefix in log output")
		}
	})
}
