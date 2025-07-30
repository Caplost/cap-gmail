package services

import (
	"crypto/tls"
	"fmt"
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/utils"
	"net"
	"net/smtp"
	"strings"
	"time"
)

type SMTPService struct {
	host     string
	port     int
	user     string
	password string
	auth     smtp.Auth
}

// NewSMTPService 创建SMTP邮件发送服务
func NewSMTPService() *SMTPService {
	cfg := config.AppConfig

	if cfg.SMTPUser == "" || cfg.SMTPPassword == "" {
		utils.LogError("SMTP配置不完整，请检查SMTP_USER和SMTP_PASSWORD环境变量", fmt.Errorf("missing SMTP credentials"))
		return nil
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)

	return &SMTPService{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		user:     cfg.SMTPUser,
		password: cfg.SMTPPassword,
		auth:     auth,
	}
}

// SendEmail 发送邮件（SMTP方式）
func (s *SMTPService) SendEmail(to, subject, body string) error {
	if s == nil {
		return fmt.Errorf("SMTP服务未初始化")
	}

	// 验证邮箱格式
	if !utils.IsValidEmail(to) {
		return fmt.Errorf("无效的收件人邮箱: %s", to)
	}

	// 构建邮件内容
	message := s.buildMessage(s.user, to, subject, body)

	// SMTP服务器地址
	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// 发送邮件
	err := s.sendWithTLS(addr, []string{to}, message)
	if err != nil {
		utils.LogError(fmt.Sprintf("SMTP发送邮件失败 to=%s", utils.MaskEmail(to)), err)
		return fmt.Errorf("SMTP发送失败: %v", err)
	}

	utils.LogInfo(fmt.Sprintf("SMTP发送邮件成功 to=%s subject=%s",
		utils.MaskEmail(to), utils.MaskSubject(subject)))
	return nil
}

// ForwardEmail SMTP方式转发邮件（标准转发格式）
func (s *SMTPService) ForwardEmail(originalFrom, originalSubject, originalBody, targetEmail string) error {
	if s == nil {
		return fmt.Errorf("SMTP服务未初始化")
	}

	// 🔧 发送前检查并强制解码邮件内容
	fmt.Printf("\n🚀 SMTP发送前解码检查:\n")
	fmt.Printf("  原始Body: %s\n", truncateSMTPString(originalBody, 100))

	// 检查是否包含编码字符
	hasEncoding := strings.Contains(originalBody, "=E") ||
		strings.Contains(originalBody, "=C") ||
		strings.Contains(originalBody, "=D")
	fmt.Printf("  检测到编码字符: %v\n", hasEncoding)

	// 如果检测到编码，强制解码
	if hasEncoding {
		decodedBody := utils.DecodeEmailContent(originalBody, "quoted-printable")
		fmt.Printf("  强制解码后Body: %s\n", truncateSMTPString(decodedBody, 100))
		fmt.Printf("  解码是否成功: %v\n", decodedBody != originalBody)
		originalBody = decodedBody
	} else {
		fmt.Printf("  无编码字符，跳过解码\n")
	}
	fmt.Printf("=====================================\n\n")

	// 使用标准的转发主题格式
	forwardSubject := fmt.Sprintf("Fwd: %s", originalSubject)

	// 构建标准转发邮件内容（保持原邮件完整性）
	forwardBody := s.buildStandardForwardContent(originalFrom, originalSubject, originalBody)

	return s.SendEmail(targetEmail, forwardSubject, forwardBody)
}

// buildMessage 构建邮件消息
func (s *SMTPService) buildMessage(from, to, subject, body string) string {
	// 基础邮件头
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}

	// 构建消息
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	return message
}

// buildStandardForwardContent 构建标准转发邮件内容（类似邮件客户端）
func (s *SMTPService) buildStandardForwardContent(originalFrom, originalSubject, originalBody string) string {
	// 标准转发格式：保持原邮件内容，添加转发头信息
	content := fmt.Sprintf(`

---------- Forwarded message ---------
From: %s
Subject: %s
Date: %s

%s`, originalFrom, originalSubject, getCurrentTime(), originalBody)

	return content
}

// sendWithTLS 使用TLS发送邮件
func (s *SMTPService) sendWithTLS(addr string, to []string, message string) error {
	// 先建立普通TCP连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("SMTP客户端创建失败: %v", err)
	}
	defer client.Quit()

	// 使用STARTTLS升级连接
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.host,
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS升级失败: %v", err)
		}
	}

	// 认证
	if s.auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(s.auth); err != nil {
				return fmt.Errorf("SMTP认证失败: %v", err)
			}
		}
	}

	// 设置发件人
	if err := client.Mail(s.user); err != nil {
		return fmt.Errorf("设置发件人失败: %v", err)
	}

	// 设置收件人
	for _, recipient := range to {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("设置收件人失败 %s: %v", recipient, err)
		}
	}

	// 发送邮件内容
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("开始数据传输失败: %v", err)
	}
	defer writer.Close()

	_, err = writer.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %v", err)
	}

	return nil
}

// GetSendLimits 获取SMTP发送限制信息（根据账户类型）
func (s *SMTPService) GetSendLimits() map[string]interface{} {
	accountType := s.detectAccountType()
	limits := config.GetAccountLimits(accountType)

	return map[string]interface{}{
		"service_type":        "SMTP",
		"account_type":        string(limits.Type),
		"account_description": limits.Description,
		"daily_limit":         limits.DailyLimit,
		"hourly_limit":        limits.HourlyLimit,
		"minute_limit":        limits.MinuteLimit,
		"concurrent_limit":    limits.ConcurrentLimit,
		"monthly_limit":       limits.MonthlyLimit,
		"recommended_use":     limits.RecommendedUse,
		"smtp_advantage":      "SMTP无秒级限制，突破Gmail API配额限制",
		"vs_gmail_api": map[string]interface{}{
			"gmail_api_limit":   "250配额/秒/用户（所有账户类型相同）",
			"smtp_advantage":    fmt.Sprintf("SMTP日限制%d封，无秒级限制", limits.DailyLimit),
			"improvement_ratio": float64(limits.DailyLimit) / 500.0, // 相对免费账户的提升倍数
		},
	}
}

// detectAccountType 检测账户类型
func (s *SMTPService) detectAccountType() config.AccountType {
	if s == nil || s.user == "" {
		return config.AccountTypeFree
	}

	return config.DetectAccountType(s.user)
}

// TestConnection 测试SMTP连接
func (s *SMTPService) TestConnection() error {
	if s == nil {
		return fmt.Errorf("SMTP服务未初始化")
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)

	// 先建立普通TCP连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("SMTP连接测试失败: %v", err)
	}
	defer conn.Close()

	// 创建SMTP客户端
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("SMTP客户端测试失败: %v", err)
	}
	defer client.Quit()

	// 使用STARTTLS升级连接
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.host,
	}

	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("STARTTLS升级失败: %v", err)
		}
	}

	// 测试认证
	if s.auth != nil {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(s.auth); err != nil {
				return fmt.Errorf("SMTP认证测试失败: %v", err)
			}
		}
	}

	utils.LogInfo("SMTP连接测试成功")
	return nil
}

// GetUpgradeRecommendations 获取账户升级建议
func (s *SMTPService) GetUpgradeRecommendations(currentDailyUsage int) map[string]interface{} {
	accountType := s.detectAccountType()
	return config.UpgradeRecommendations(accountType, currentDailyUsage)
}

// GetOptimalStrategy 获取最优发送策略
func (s *SMTPService) GetOptimalStrategy() map[string]interface{} {
	accountType := s.detectAccountType()
	return config.GetOptimalSendingStrategy(accountType)
}

// getCurrentTime 获取当前时间字符串
func getCurrentTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// truncateSMTPString 截断字符串用于SMTP日志显示
func truncateSMTPString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
