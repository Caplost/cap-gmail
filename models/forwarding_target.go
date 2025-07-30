package models

import (
	"time"

	"gorm.io/gorm"
)

// ForwardingTarget 转发对象模型
type ForwardingTarget struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	Name      string    `json:"name" gorm:"not null;comment:转发对象名字"`
	Email     string    `json:"email" gorm:"not null;comment:转发邮箱地址"`
	Keywords  string    `json:"keywords" gorm:"comment:关联的关键字，逗号分隔"`
	IsActive  bool      `json:"is_active" gorm:"default:true;comment:是否启用"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ForwardingTarget) TableName() string {
	return "forwarding_targets"
}

// CreateForwardingTarget 创建转发对象
func CreateForwardingTarget(db *gorm.DB, target *ForwardingTarget) error {
	return db.Create(target).Error
}

// GetForwardingTargets 获取所有转发对象
func GetForwardingTargets(db *gorm.DB) ([]ForwardingTarget, error) {
	var targets []ForwardingTarget
	err := db.Where("is_active = ?", true).Find(&targets).Error
	return targets, err
}

// GetForwardingTargetByID 根据ID获取转发对象
func GetForwardingTargetByID(db *gorm.DB, id uint) (*ForwardingTarget, error) {
	var target ForwardingTarget
	err := db.First(&target, id).Error
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// GetForwardingTargetByName 根据名字获取转发对象
func GetForwardingTargetByName(db *gorm.DB, name string) (*ForwardingTarget, error) {
	var target ForwardingTarget
	err := db.Where("name = ? AND is_active = ?", name, true).First(&target).Error
	if err != nil {
		return nil, err
	}
	return &target, nil
}

// UpdateForwardingTarget 更新转发对象
func UpdateForwardingTarget(db *gorm.DB, target *ForwardingTarget) error {
	return db.Save(target).Error
}

// DeleteForwardingTarget 软删除转发对象
func DeleteForwardingTarget(db *gorm.DB, id uint) error {
	return db.Model(&ForwardingTarget{}).Where("id = ?", id).Update("is_active", false).Error
}
