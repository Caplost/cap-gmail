package utils

import (
	"log"
	"os"
)

var Logger *log.Logger

// InitLogger 初始化日志器
func InitLogger() {
	Logger = log.New(os.Stdout, "[Gmail-Forwarding] ", log.LstdFlags|log.Lshortfile)
}

// LogInfo 记录信息日志
func LogInfo(msg string) {
	Logger.Printf("[INFO] %s", msg)
}

// LogError 记录错误日志
func LogError(msg string, err error) {
	if err != nil {
		Logger.Printf("[ERROR] %s: %v", msg, err)
	} else {
		Logger.Printf("[ERROR] %s", msg)
	}
}

// LogWarn 记录警告日志
func LogWarn(msg string) {
	Logger.Printf("[WARN] %s", msg)
}
