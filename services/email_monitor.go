package services

import (
	"fmt"
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/utils"
	"sync"
	"time"

	"gorm.io/gorm"
)

// EmailMonitor 邮件监听服务
type EmailMonitor struct {
	db                *gorm.DB
	forwardingService *ForwardingService
	imapService       *IMAPService
	isRunning         bool
	stopChan          chan bool
	mu                sync.Mutex
}

// NewEmailMonitor 创建邮件监听服务
func NewEmailMonitor(db *gorm.DB) (*EmailMonitor, error) {
	// 创建转发服务
	forwardingService := NewForwardingService(db)
	if forwardingService == nil {
		return nil, fmt.Errorf("创建转发服务失败")
	}

	// 创建IMAP服务
	imapService := NewIMAPService()
	if imapService == nil {
		return nil, fmt.Errorf("创建IMAP服务失败")
	}

	return &EmailMonitor{
		db:                db,
		forwardingService: forwardingService,
		imapService:       imapService,
		isRunning:         false,
		stopChan:          make(chan bool),
	}, nil
}

// Start 启动邮件监听
func (em *EmailMonitor) Start() {
	em.mu.Lock()
	defer em.mu.Unlock()

	if em.isRunning {
		utils.LogInfo("邮件监听服务已在运行中")
		return
	}

	em.isRunning = true
	utils.LogInfo("🚀 启动邮件监听服务...")

	go em.monitorLoop()
}

// Stop 停止邮件监听
func (em *EmailMonitor) Stop() {
	em.mu.Lock()
	defer em.mu.Unlock()

	if !em.isRunning {
		return
	}

	utils.LogInfo("🛑 停止邮件监听服务...")
	em.isRunning = false
	em.stopChan <- true
}

// IsRunning 检查是否在运行
func (em *EmailMonitor) IsRunning() bool {
	em.mu.Lock()
	defer em.mu.Unlock()
	return em.isRunning
}

// monitorLoop 监听循环
func (em *EmailMonitor) monitorLoop() {
	utils.LogInfo("📧 邮件监听循环开始...")

	// 获取检查间隔，默认30秒
	interval := 30 * time.Second
	if config.AppConfig.EmailCheckInterval > 0 {
		interval = time.Duration(config.AppConfig.EmailCheckInterval) * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-em.stopChan:
			utils.LogInfo("📧 邮件监听循环停止")
			return
		case <-ticker.C:
			em.checkAndProcessEmails()
		}
	}
}

// checkAndProcessEmails 检查并处理邮件
func (em *EmailMonitor) checkAndProcessEmails() {
	utils.LogInfo("🔍 检查新邮件...")

	// 获取新邮件
	emails, err := em.imapService.GetUnreadEmails()
	if err != nil {
		utils.LogError("获取邮件失败", err)
		return
	}

	if len(emails) == 0 {
		utils.LogInfo("📭 暂无新邮件")
		return
	}

	utils.LogInfo(fmt.Sprintf("📬 发现 %d 封新邮件，开始处理...", len(emails)))

	// 处理每封邮件
	for i, email := range emails {
		utils.LogInfo(fmt.Sprintf("📧 处理邮件 %d/%d: %s",
			i+1, len(emails), utils.MaskSubject(email.Subject)))

		err := em.forwardingService.processEmailWithPriority(email)
		if err != nil {
			utils.LogError(fmt.Sprintf("处理邮件失败 [%s]", utils.MaskSubject(email.Subject)), err)
		} else {
			utils.LogInfo(fmt.Sprintf("✅ 邮件处理完成 [%s]", utils.MaskSubject(email.Subject)))
		}

		// 避免处理过快，添加小延迟
		time.Sleep(1 * time.Second)
	}

	utils.LogInfo(fmt.Sprintf("🎉 本轮邮件处理完成，处理了 %d 封邮件", len(emails)))
}

// GetStatus 获取监听状态
func (em *EmailMonitor) GetStatus() map[string]interface{} {
	em.mu.Lock()
	defer em.mu.Unlock()

	return map[string]interface{}{
		"running":        em.isRunning,
		"imap_host":      config.AppConfig.IMAPHost,
		"imap_user":      utils.MaskEmail(config.AppConfig.IMAPUser),
		"check_interval": config.AppConfig.EmailCheckInterval,
	}
}
