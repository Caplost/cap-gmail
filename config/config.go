package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config 应用配置结构
type Config struct {
	// 服务器配置
	ServerPort string

	// 数据库配置
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Gmail API配置
	GmailCredentialsPath string
	GmailTokenPath       string

	// IMAP配置
	IMAPHost     string
	IMAPPort     int
	IMAPUser     string
	IMAPPassword string

	// SMTP配置
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string

	// 邮件接收策略
	PreferIMAP   bool
	EnableHybrid bool

	// 邮件发送策略（保留兼容性）
	PreferSMTP       bool
	EnableSMTPHybrid bool

	// 邮件监听配置
	EmailCheckInterval int  // 检查邮件间隔(秒)
	AutoMonitor        bool // 是否启用自动监听
}

var AppConfig *Config

func LoadConfig() {
	// 加载 .env 文件
	if err := godotenv.Load(); err != nil {
		log.Println("未找到 .env 文件，使用环境变量")
	}

	// 解析端口配置
	smtpPort := getEnvInt("SMTP_PORT", 587)
	imapPort := getEnvInt("IMAP_PORT", 993)

	// 解析布尔配置
	preferIMAP := parseBool(os.Getenv("PREFER_IMAP"), false)
	enableHybrid := parseBool(os.Getenv("ENABLE_HYBRID"), true)
	preferSMTP := parseBool(os.Getenv("PREFER_SMTP"), true)
	enableSMTPHybrid := parseBool(os.Getenv("ENABLE_SMTP_HYBRID"), true)

	AppConfig = &Config{
		// 数据库配置
		DBHost:     getEnvOrDefault("DB_HOST", "localhost"),
		DBPort:     getEnvOrDefault("DB_PORT", "3306"),
		DBUser:     getEnvOrDefault("DB_USER", "root"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     getEnvOrDefault("DB_NAME", "gmail_forwarding"),

		// Gmail API 配置
		GmailCredentialsPath: getEnvOrDefault("GMAIL_CREDENTIALS_PATH", "credentials.json"),
		GmailTokenPath:       getEnvOrDefault("GMAIL_TOKEN_PATH", "token.json"),

		// SMTP 配置
		SMTPHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     smtpPort,
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),

		// IMAP 配置
		IMAPHost:     getEnvOrDefault("IMAP_HOST", "imap.gmail.com"),
		IMAPPort:     imapPort,
		IMAPUser:     os.Getenv("IMAP_USER"),
		IMAPPassword: os.Getenv("IMAP_PASSWORD"),

		// 邮件接收策略配置
		PreferIMAP:   preferIMAP,
		EnableHybrid: enableHybrid,

		// 邮件发送策略配置
		PreferSMTP:       preferSMTP,
		EnableSMTPHybrid: enableSMTPHybrid,

		// 邮件监听配置
		EmailCheckInterval: getEnvInt("EMAIL_CHECK_INTERVAL", 30),
		AutoMonitor:        parseBool(os.Getenv("AUTO_MONITOR"), true),

		// 服务器配置
		ServerPort: getEnvOrDefault("SERVER_PORT", "8080"),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	result, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return result
}

// getEnvInt 获取整数环境变量
func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return result
}
