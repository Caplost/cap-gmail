package services

import (
	"fmt"
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/models"
	"gmail-forwarding-system/utils"
	"strings"
	"time"

	"gorm.io/gorm"
)

// EmailReceiver 邮件接收接口
type EmailReceiver interface {
	GetUnreadEmails() ([]*EmailData, error)
	MarkAsRead(emailID string) error
}

// ForwardingService 转发服务（支持混合接收模式）
type ForwardingService struct {
	// 数据库连接
	db *gorm.DB

	// 邮件接收服务
	gmailService *GmailService // Gmail API接收
	imapService  *IMAPService  // IMAP接收

	// 邮件发送服务
	smtpService *SMTPService // SMTP发送

	// 其他服务
	emailParser *EmailParser
}

// NewForwardingService 创建转发服务实例
func NewForwardingService(db *gorm.DB) *ForwardingService {
	fs := &ForwardingService{
		db:          db,
		emailParser: NewEmailParser(db),
	}

	// 初始化Gmail API服务（用于接收，如果IMAP不可用时的备选）
	if gmailService, err := NewGmailService(); err == nil && gmailService != nil {
		fs.gmailService = gmailService
		utils.LogInfo("Gmail API服务初始化成功")
	} else {
		utils.LogWarn("Gmail API服务初始化失败")
	}

	// 初始化IMAP服务（用于接收，优先选择如果配置了的话）
	if imapService := NewIMAPService(); imapService != nil {
		fs.imapService = imapService
		utils.LogInfo("IMAP服务初始化成功")
	} else {
		utils.LogWarn("IMAP服务初始化失败")
	}

	// 初始化SMTP服务（用于发送）
	if smtpService := NewSMTPService(); smtpService != nil {
		fs.smtpService = smtpService
		utils.LogInfo("SMTP服务初始化成功")
	} else {
		utils.LogWarn("SMTP服务初始化失败")
	}

	// 检查至少有一个可用的接收服务
	if fs.gmailService == nil && fs.imapService == nil {
		utils.LogError("无可用的邮件接收服务", nil)
		return nil
	}

	// 检查至少有一个可用的发送服务
	if fs.smtpService == nil && fs.gmailService == nil {
		utils.LogError("无可用的邮件发送服务", nil)
		return nil
	}

	return fs
}

// ProcessEmails 处理邮件（混合接收模式）
func (fs *ForwardingService) ProcessEmails() error {
	// 选择邮件接收服务
	receiver := fs.selectEmailReceiver()
	if receiver == nil {
		return fmt.Errorf("无可用的邮件接收服务")
	}

	// 获取未读邮件
	emails, err := receiver.GetUnreadEmails()
	if err != nil {
		return fmt.Errorf("获取未读邮件失败: %v", err)
	}

	if len(emails) == 0 {
		utils.LogInfo("没有新邮件需要处理")
		return nil
	}

	utils.LogInfo(fmt.Sprintf("开始处理 %d 封邮件", len(emails)))

	for _, email := range emails {
		if err := fs.processEmailWithPriority(email); err != nil {
			utils.LogError(fmt.Sprintf("处理邮件失败: %s", utils.MaskSubject(email.Subject)), err)
		}

		// 标记邮件为已读
		if err := receiver.MarkAsRead(email.ID); err != nil {
			utils.LogWarn(fmt.Sprintf("标记邮件已读失败: %v", err))
		}
	}

	return nil
}

// selectEmailReceiver 选择邮件接收服务（IMAP优先策略）
func (fs *ForwardingService) selectEmailReceiver() EmailReceiver {
	// 策略1：优先使用IMAP（如果配置且可用）
	if config.AppConfig.PreferIMAP && fs.imapService != nil {
		utils.LogInfo("使用IMAP接收邮件")
		return fs.imapService
	}

	// 策略2：使用Gmail API（如果可用）
	if fs.gmailService != nil {
		utils.LogInfo("使用Gmail API接收邮件")
		return fs.gmailService
	}

	// 策略3：混合模式下的备选
	if config.AppConfig.EnableHybrid {
		if fs.imapService != nil {
			utils.LogInfo("混合模式：使用IMAP接收邮件")
			return fs.imapService
		}
		if fs.gmailService != nil {
			utils.LogInfo("混合模式：使用Gmail API接收邮件")
			return fs.gmailService
		}
	}

	return nil
}

// processEmailWithPriority 处理单封邮件（使用发送优先级策略）
func (fs *ForwardingService) processEmailWithPriority(email *EmailData) error {
	// 验证邮件MessageID
	if email.MessageID == "" {
		utils.LogWarn("邮件缺少MessageID，跳过处理")
		return nil
	}

	// 解析邮件主题，匹配转发目标
	parsedInfo, err := fs.emailParser.ParseEmailSubject(email.Subject)
	if err != nil || !parsedInfo.ShouldForward || parsedInfo.Target == nil {
		utils.LogInfo(fmt.Sprintf("邮件主题无匹配的转发目标: %s", utils.MaskSubject(email.Subject)))

		// 记录跳过日志（如果没有记录过）
		_, existingLog, _ := fs.checkEmailAlreadyProcessed(email.MessageID, "")
		if existingLog == nil {
			fs.logEmailSkipped(email, "无匹配的转发目标")
		}
		return nil
	}

	targets := []*models.ForwardingTarget{parsedInfo.Target}
	if len(targets) == 0 {
		utils.LogInfo(fmt.Sprintf("邮件主题无匹配的转发目标: %s", utils.MaskSubject(email.Subject)))

		// 记录跳过日志（如果没有记录过）
		_, existingLog, _ := fs.checkEmailAlreadyProcessed(email.MessageID, "")
		if existingLog == nil {
			fs.logEmailSkipped(email, "无匹配的转发目标")
		}
		return nil
	}

	utils.LogInfo(fmt.Sprintf("邮件匹配到 %d 个转发目标 MessageID=%s", len(targets), utils.MaskSubject(email.MessageID)))

	// 逐个转发到匹配的目标
	for _, target := range targets {
		// 检查是否已经处理过这封邮件到这个目标
		alreadyProcessed, existingLog, err := fs.checkEmailAlreadyProcessed(email.MessageID, target.Email)
		if err != nil {
			utils.LogError(fmt.Sprintf("检查邮件处理状态失败: %v", err), err)
			continue
		}

		if alreadyProcessed {
			utils.LogInfo(fmt.Sprintf("邮件已处理，跳过转发 MessageID=%s Target=%s",
				utils.MaskSubject(email.MessageID), utils.MaskEmail(target.Email)))
			continue
		}

		// 转发邮件
		if err := fs.forwardEmailWithPriority(email, target); err != nil {
			// 记录失败日志或更新现有记录
			if existingLog != nil {
				fs.updateEmailLogResult(existingLog.ID, false, err.Error())
			} else {
				fs.logEmailResult(email, target, false, err.Error())
			}
			utils.LogError(fmt.Sprintf("转发邮件失败 to=%s", utils.MaskEmail(target.Email)), err)
		} else {
			// 记录成功日志或更新现有记录
			if existingLog != nil {
				fs.updateEmailLogResult(existingLog.ID, true, "")
			} else {
				fs.logEmailResult(email, target, true, "")
			}
			utils.LogInfo(fmt.Sprintf("邮件转发成功 to=%s", utils.MaskEmail(target.Email)))
		}
	}

	return nil
}

// forwardEmailWithPriority 转发邮件（SMTP优先，API备选）
func (fs *ForwardingService) forwardEmailWithPriority(email *EmailData, target *models.ForwardingTarget) error {
	// 📝 打印转发前的邮件内容到控制台
	fmt.Printf("\n🔍 转发前邮件内容检查:\n")
	fmt.Printf("  主题: %s\n", email.Subject)
	fmt.Printf("  发件人: %s\n", email.From)
	fmt.Printf("  转发目标: %s (%s)\n", target.Name, target.Email)
	fmt.Printf("  原始Body长度: %d 字符\n", len(email.Body))
	fmt.Printf("  原始Body前200字符: %s\n", truncateString(email.Body, 200))
	if containsEncodingChars(email.Body) {
		fmt.Printf("  ⚠️  检测到编码字符 (=E开头)\n")
	} else {
		fmt.Printf("  ✅ 内容看起来已解码\n")
	}
	fmt.Printf("  完整Body内容:\n%s\n", email.Body)
	fmt.Printf("==========================================\n\n")

	// 📝 注释：解码逻辑已移至SMTP/Gmail API发送前，避免双重解码

	// 策略1：优先使用SMTP（如果可用且配置为优先）
	if fs.smtpService != nil && config.AppConfig.PreferSMTP {
		utils.LogInfo(fmt.Sprintf("使用SMTP转发邮件 to=%s", utils.MaskEmail(target.Email)))
		err := fs.smtpService.ForwardEmail(email.From, email.Subject, email.Body, target.Email)
		if err == nil {
			return nil // SMTP发送成功
		}
		// SMTP失败，记录日志但继续尝试备选方案
		utils.LogWarn(fmt.Sprintf("SMTP转发失败，尝试Gmail API备选: %v", err))
	}

	// 策略2：使用Gmail API发送（作为备选或主要方式）
	if fs.gmailService != nil {
		utils.LogInfo(fmt.Sprintf("使用Gmail API发送邮件 to=%s", utils.MaskEmail(target.Email)))
		return fs.gmailService.ForwardEmail(email, target.Email)
	}

	return fmt.Errorf("无可用的邮件发送服务")
}

// truncateString 截断字符串用于显示
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// containsEncodingChars 检查是否包含编码字符
func containsEncodingChars(content string) bool {
	return strings.Contains(content, "=E") ||
		strings.Contains(content, "=C") ||
		strings.Contains(content, "=D")
}

// buildForwardContent 构建转发邮件内容
func (fs *ForwardingService) buildForwardContent(email *EmailData) string {
	return fmt.Sprintf(`
========== 转发邮件 ==========
原发件人: %s
原主题: %s
接收时间: %s
转发时间: %s

========== 原邮件内容 ==========
%s
`,
		email.From,
		email.Subject,
		email.ReceivedTime.Format("2006-01-02 15:04:05"),
		getCurrentTimeString(),
		email.Body,
	)
}

// logEmailResult 记录邮件处理结果
func (fs *ForwardingService) logEmailResult(email *EmailData, target *models.ForwardingTarget, success bool, errorMsg string) {
	emailLog := &models.EmailLog{
		MessageID:   email.MessageID,
		FromEmail:   email.From,
		ForwardedTo: target.Email,
		Subject:     email.Subject,
		TargetName:  target.Name,
		Status:      models.EmailStatus(getLogStatus(success)),
		ErrorMsg:    errorMsg,
	}

	if err := models.CreateEmailLog(fs.db, emailLog); err != nil {
		utils.LogError("保存邮件日志失败", err)
	}
}

// GetEmailLogs 获取邮件日志
func (fs *ForwardingService) GetEmailLogs(page, pageSize int) ([]models.EmailLog, int64, error) {
	// 如果服务未初始化，返回空结果（用于测试）
	if fs == nil {
		return []models.EmailLog{}, 0, nil
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	// 查询日志
	var logs []models.EmailLog
	var total int64

	if err := fs.db.Model(&models.EmailLog{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计邮件日志失败: %v", err)
	}

	if err := fs.db.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询邮件日志失败: %v", err)
	}

	return logs, total, nil
}

// GetServiceStatus 获取服务状态
func (fs *ForwardingService) GetServiceStatus() map[string]interface{} {
	status := map[string]interface{}{
		"receiving_services": map[string]interface{}{},
		"sending_services":   map[string]interface{}{},
		"strategy":           fs.getStrategyInfo(),
	}

	// 接收服务状态
	if fs.imapService != nil {
		status["receiving_services"].(map[string]interface{})["imap"] = fs.imapService.GetServiceInfo()
	}
	if fs.gmailService != nil {
		status["receiving_services"].(map[string]interface{})["gmail_api"] = fs.gmailService.GetServiceInfo()
	}

	// 发送服务状态
	if fs.smtpService != nil {
		status["sending_services"].(map[string]interface{})["smtp"] = map[string]interface{}{
			"service_type": "SMTP",
			"host":         "smtp.gmail.com",
			"status":       "active",
			"auth_method":  "app_password",
		}
	}
	if fs.gmailService != nil {
		status["sending_services"].(map[string]interface{})["gmail_api"] = fs.gmailService.GetServiceInfo()
	}

	return status
}

// getStrategyInfo 获取策略信息
func (fs *ForwardingService) getStrategyInfo() map[string]interface{} {
	return map[string]interface{}{
		"prefer_imap":        config.AppConfig.PreferIMAP,
		"prefer_smtp":        config.AppConfig.PreferSMTP,
		"enable_hybrid":      config.AppConfig.EnableHybrid,
		"enable_smtp_hybrid": config.AppConfig.EnableSMTPHybrid,
		"current_receiver":   fs.getCurrentReceiverType(),
		"current_sender":     fs.getCurrentSenderType(),
	}
}

// getCurrentReceiverType 获取当前接收服务类型
func (fs *ForwardingService) getCurrentReceiverType() string {
	receiver := fs.selectEmailReceiver()
	if receiver == fs.imapService {
		return "IMAP"
	}
	if receiver == fs.gmailService {
		return "Gmail_API"
	}
	return "None"
}

// getCurrentSenderType 获取当前发送服务类型
func (fs *ForwardingService) getCurrentSenderType() string {
	if config.AppConfig.PreferSMTP && fs.smtpService != nil {
		return "SMTP_Primary"
	}
	if fs.gmailService != nil {
		return "Gmail_API_Fallback"
	}
	return "None"
}

// getCurrentTimeString 获取当前时间字符串
func getCurrentTimeString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// getLogStatus 将布尔成功状态转换为字符串状态
func getLogStatus(success bool) string {
	if success {
		return "forwarded"
	}
	return "failed"
}

// checkEmailAlreadyProcessed 检查邮件是否已经处理过
func (fs *ForwardingService) checkEmailAlreadyProcessed(messageID string, targetEmail string) (bool, *models.EmailLog, error) {
	// 查询是否存在相同MessageID和目标邮箱的处理记录
	existingLog, err := models.GetEmailLogByMessageIDAndTarget(fs.db, messageID, targetEmail)
	if err != nil {
		// 如果是记录不存在的错误，说明没有处理过
		if err.Error() == "record not found" || strings.Contains(err.Error(), "record not found") {
			return false, nil, nil
		}
		return false, nil, fmt.Errorf("查询邮件处理记录失败: %v", err)
	}

	// 检查处理状态
	switch existingLog.Status {
	case models.StatusForwarded:
		utils.LogInfo(fmt.Sprintf("邮件已成功转发过 MessageID=%s Target=%s",
			utils.MaskSubject(messageID), utils.MaskEmail(targetEmail)))
		return true, existingLog, nil
	case models.StatusFailed:
		utils.LogInfo(fmt.Sprintf("邮件之前转发失败，将重试 MessageID=%s Target=%s",
			utils.MaskSubject(messageID), utils.MaskEmail(targetEmail)))
		return false, existingLog, nil
	case models.StatusSkipped:
		utils.LogInfo(fmt.Sprintf("邮件之前被跳过，将重新处理 MessageID=%s Target=%s",
			utils.MaskSubject(messageID), utils.MaskEmail(targetEmail)))
		return false, existingLog, nil
	case models.StatusPending:
		utils.LogInfo(fmt.Sprintf("邮件正在处理中，将重试 MessageID=%s Target=%s",
			utils.MaskSubject(messageID), utils.MaskEmail(targetEmail)))
		return false, existingLog, nil
	}

	return false, existingLog, nil
}

// isDuplicateForwardAttempt 检查是否是重复的转发尝试
func (fs *ForwardingService) isDuplicateForwardAttempt(messageID string, targetEmail string) bool {
	// 查询相同MessageID和目标邮箱的成功转发记录
	var count int64
	fs.db.Model(&models.EmailLog{}).
		Where("message_id = ? AND forwarded_to = ? AND status = ?",
			messageID, targetEmail, models.StatusForwarded).
		Count(&count)

	return count > 0
}

// logEmailSkipped 记录跳过的邮件
func (fs *ForwardingService) logEmailSkipped(email *EmailData, reason string) {
	emailLog := &models.EmailLog{
		MessageID:   email.MessageID,
		Subject:     email.Subject,
		FromEmail:   email.From,
		ToEmail:     "", // 没有收件人
		ForwardedTo: "",
		Keyword:     "",
		TargetName:  "",
		Status:      models.StatusSkipped,
		ErrorMsg:    reason,
	}

	if err := models.CreateEmailLog(fs.db, emailLog); err != nil {
		utils.LogError("保存跳过邮件日志失败", err)
	}
}

// updateEmailLogResult 更新邮件日志结果
func (fs *ForwardingService) updateEmailLogResult(logID uint, success bool, errorMsg string) {
	var status models.EmailStatus
	if success {
		status = models.StatusForwarded
	} else {
		status = models.StatusFailed
	}

	if err := models.UpdateEmailLogStatus(fs.db, logID, status, errorMsg); err != nil {
		utils.LogError("更新邮件日志状态失败", err)
	}
}
