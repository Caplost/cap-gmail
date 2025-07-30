package models

import (
	"time"

	"gorm.io/gorm"
)

// EmailStatus 邮件处理状态
type EmailStatus string

const (
	StatusPending   EmailStatus = "pending"
	StatusForwarded EmailStatus = "forwarded"
	StatusFailed    EmailStatus = "failed"
	StatusSkipped   EmailStatus = "skipped" // 不匹配关键字时跳过
)

// EmailLog 邮件处理日志模型
type EmailLog struct {
	ID          uint        `json:"id" gorm:"primarykey"`
	MessageID   string      `json:"message_id" gorm:"not null;comment:邮件唯一标识"`
	Subject     string      `json:"subject" gorm:"comment:邮件主题"`
	FromEmail   string      `json:"from_email" gorm:"comment:发件人邮箱"`
	ToEmail     string      `json:"to_email" gorm:"comment:收件人邮箱"`
	ForwardedTo string      `json:"forwarded_to" gorm:"comment:转发到的邮箱"`
	Keyword     string      `json:"keyword" gorm:"comment:匹配的关键字"`
	TargetName  string      `json:"target_name" gorm:"comment:转发对象名字"`
	Status      EmailStatus `json:"status" gorm:"default:pending;comment:处理状态"`
	ErrorMsg    string      `json:"error_msg" gorm:"comment:错误信息"`
	ProcessedAt *time.Time  `json:"processed_at" gorm:"comment:处理时间"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// 表约束：同一邮件到同一目标只能有一条记录
func (EmailLog) BeforeCreate(tx *gorm.DB) error {
	return nil
}

// TableName 指定表名
func (EmailLog) TableName() string {
	return "email_logs"
}

// CreateEmailLog 创建邮件日志
func CreateEmailLog(db *gorm.DB, log *EmailLog) error {
	return db.Create(log).Error
}

// GetEmailLogs 获取邮件日志列表
func GetEmailLogs(db *gorm.DB, page, pageSize int) ([]EmailLog, int64, error) {
	var logs []EmailLog
	var total int64

	// 计算总数
	db.Model(&EmailLog{}).Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error

	return logs, total, err
}

// GetEmailLogByMessageID 根据MessageID查询日志（返回第一条）
func GetEmailLogByMessageID(db *gorm.DB, messageID string) (*EmailLog, error) {
	var log EmailLog
	err := db.Where("message_id = ?", messageID).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetEmailLogByMessageIDAndTarget 根据MessageID和目标邮箱查询日志
func GetEmailLogByMessageIDAndTarget(db *gorm.DB, messageID, targetEmail string) (*EmailLog, error) {
	var log EmailLog
	err := db.Where("message_id = ? AND forwarded_to = ?", messageID, targetEmail).First(&log).Error
	if err != nil {
		return nil, err
	}
	return &log, nil
}

// GetEmailLogsByMessageID 根据MessageID查询所有相关日志
func GetEmailLogsByMessageID(db *gorm.DB, messageID string) ([]EmailLog, error) {
	var logs []EmailLog
	err := db.Where("message_id = ?", messageID).Find(&logs).Error
	return logs, err
}

// UpdateEmailLogStatus 更新邮件日志状态
func UpdateEmailLogStatus(db *gorm.DB, id uint, status EmailStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status":       status,
		"processed_at": time.Now(),
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	return db.Model(&EmailLog{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateEmailLogForwarded 更新转发成功状态
func UpdateEmailLogForwarded(db *gorm.DB, id uint, forwardedTo string) error {
	now := time.Now()
	return db.Model(&EmailLog{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       StatusForwarded,
		"forwarded_to": forwardedTo,
		"processed_at": &now,
	}).Error
}
