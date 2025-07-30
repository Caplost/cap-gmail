package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	// 自动迁移
	db.AutoMigrate(&ForwardingTarget{})
	return db
}

func TestForwardingTargetModel(t *testing.T) {
	t.Run("CreateForwardingTarget", func(t *testing.T) {
		db := setupTestDB()

		target := &ForwardingTarget{
			Name:     "测试用户",
			Email:    "test@example.com",
			Keywords: "客服,售后",
			IsActive: true,
		}

		err := CreateForwardingTarget(db, target)
		if err != nil {
			t.Errorf("CreateForwardingTarget failed: %v", err)
		}

		if target.ID == 0 {
			t.Error("Expected ID to be set after creation")
		}
	})

	t.Run("GetForwardingTargets", func(t *testing.T) {
		db := setupTestDB()

		// 首先确认数据库是空的
		initialResult, err := GetForwardingTargets(db)
		if err != nil {
			t.Errorf("Failed to get initial targets: %v", err)
		}
		if len(initialResult) != 0 {
			t.Errorf("Expected empty database, got %d targets", len(initialResult))
		}

		// 插入活跃用户
		activeTargets := []*ForwardingTarget{
			{Name: "用户1", Email: "user1@example.com", Keywords: "客服", IsActive: true},
			{Name: "用户2", Email: "user2@example.com", Keywords: "售后", IsActive: true},
		}

		for _, target := range activeTargets {
			CreateForwardingTarget(db, target)
		}

		// 插入非活跃用户 - 由于GORM default:true，需要先创建再更新
		inactiveTarget := &ForwardingTarget{
			Name:     "用户3",
			Email:    "user3@example.com",
			Keywords: "技术",
			IsActive: true, // 先设置为true创建，然后更新为false
		}
		CreateForwardingTarget(db, inactiveTarget)

		// 手动更新为false以避免GORM默认值影响
		db.Model(inactiveTarget).Update("is_active", false)

		// 手动验证数据库状态
		var allTargets []ForwardingTarget
		db.Find(&allTargets)
		t.Logf("Total targets in database: %d", len(allTargets))
		for i, target := range allTargets {
			t.Logf("Target %d: Name=%s, Email=%s, Active=%v", i, target.Name, target.Email, target.IsActive)
		}

		// 测试获取活跃用户
		result, err := GetForwardingTargets(db)
		if err != nil {
			t.Errorf("GetForwardingTargets failed: %v", err)
		}

		// 应该只返回活跃用户（2个）
		if len(result) != 2 {
			t.Errorf("Expected 2 active targets, got %d", len(result))
			// 打印详细信息用于调试
			for i, target := range result {
				t.Logf("Target %d: Name=%s, Email=%s, Active=%v", i, target.Name, target.Email, target.IsActive)
			}
		}
	})

	t.Run("GetForwardingTargetByName", func(t *testing.T) {
		db := setupTestDB()

		// 插入测试用户
		target := &ForwardingTarget{
			Name:     "用户1",
			Email:    "user1@example.com",
			Keywords: "客服",
			IsActive: true,
		}
		CreateForwardingTarget(db, target)

		// 测试查找存在的用户
		found, err := GetForwardingTargetByName(db, "用户1")
		if err != nil {
			t.Errorf("GetForwardingTargetByName failed: %v", err)
		}

		if found.Email != "user1@example.com" {
			t.Errorf("Expected email user1@example.com, got %s", found.Email)
		}

		// 测试不存在的用户
		_, err = GetForwardingTargetByName(db, "不存在的用户")
		if err == nil {
			t.Error("Expected error for non-existent user")
		}
	})

	t.Run("UpdateForwardingTarget", func(t *testing.T) {
		db := setupTestDB()

		// 创建测试用户
		target := &ForwardingTarget{
			Name:     "用户1",
			Email:    "user1@example.com",
			Keywords: "客服",
			IsActive: true,
		}
		CreateForwardingTarget(db, target)

		// 更新用户
		target.Keywords = "客服,售后,投诉"
		err := UpdateForwardingTarget(db, target)
		if err != nil {
			t.Errorf("UpdateForwardingTarget failed: %v", err)
		}

		// 验证更新
		updated, _ := GetForwardingTargetByID(db, target.ID)
		if updated.Keywords != "客服,售后,投诉" {
			t.Errorf("Expected updated keywords, got %s", updated.Keywords)
		}
	})

	t.Run("DeleteForwardingTarget", func(t *testing.T) {
		db := setupTestDB()

		// 创建测试用户
		target := &ForwardingTarget{
			Name:     "用户1",
			Email:    "user1@example.com",
			Keywords: "客服",
			IsActive: true,
		}
		CreateForwardingTarget(db, target)

		// 删除用户
		err := DeleteForwardingTarget(db, target.ID)
		if err != nil {
			t.Errorf("DeleteForwardingTarget failed: %v", err)
		}

		// 验证软删除
		_, err = GetForwardingTargetByName(db, "用户1")
		if err == nil {
			t.Error("Expected error after soft delete")
		}
	})
}

func TestForwardingTargetValidation(t *testing.T) {
	db := setupTestDB()

	t.Run("UniqueEmailConstraint", func(t *testing.T) {
		target1 := &ForwardingTarget{
			Name:  "用户1",
			Email: "same@example.com",
		}
		target2 := &ForwardingTarget{
			Name:  "用户2",
			Email: "same@example.com",
		}

		err1 := CreateForwardingTarget(db, target1)
		if err1 != nil {
			t.Errorf("First creation should succeed: %v", err1)
		}

		err2 := CreateForwardingTarget(db, target2)
		if err2 == nil {
			t.Error("Second creation with same email should fail")
		}
	})

	t.Run("TableName", func(t *testing.T) {
		target := ForwardingTarget{}
		tableName := target.TableName()
		if tableName != "forwarding_targets" {
			t.Errorf("Expected table name 'forwarding_targets', got %s", tableName)
		}
	})
}
