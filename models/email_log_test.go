package models

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupEmailLogTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	db.AutoMigrate(&EmailLog{})
	return db
}

func TestEmailLogModel(t *testing.T) {
	db := setupEmailLogTestDB()

	t.Run("CreateEmailLog", func(t *testing.T) {
		log := &EmailLog{
			MessageID:  "msg123",
			Subject:    "客服 - 张三",
			FromEmail:  "sender@example.com",
			ToEmail:    "receiver@gmail.com",
			Keyword:    "客服",
			TargetName: "张三",
			Status:     StatusPending,
		}

		err := CreateEmailLog(db, log)
		if err != nil {
			t.Errorf("CreateEmailLog failed: %v", err)
		}

		if log.ID == 0 {
			t.Error("Expected ID to be set after creation")
		}
	})

	t.Run("GetEmailLogByMessageID", func(t *testing.T) {
		log, err := GetEmailLogByMessageID(db, "msg123")
		if err != nil {
			t.Errorf("GetEmailLogByMessageID failed: %v", err)
		}

		if log.Subject != "客服 - 张三" {
			t.Errorf("Expected subject '客服 - 张三', got %s", log.Subject)
		}
	})

	t.Run("GetEmailLogs", func(t *testing.T) {
		// 插入更多测试数据
		logs := []EmailLog{
			{MessageID: "msg001", Subject: "测试1", Status: StatusForwarded},
			{MessageID: "msg002", Subject: "测试2", Status: StatusFailed},
			{MessageID: "msg003", Subject: "测试3", Status: StatusSkipped},
		}

		for _, log := range logs {
			CreateEmailLog(db, &log)
		}

		// 测试分页查询
		result, total, err := GetEmailLogs(db, 1, 10)
		if err != nil {
			t.Errorf("GetEmailLogs failed: %v", err)
		}

		if total < 4 { // 至少有4条记录（包括前面创建的）
			t.Errorf("Expected at least 4 total logs, got %d", total)
		}

		if len(result) == 0 {
			t.Error("Expected some results")
		}
	})

	t.Run("UpdateEmailLogStatus", func(t *testing.T) {
		log, _ := GetEmailLogByMessageID(db, "msg123")

		err := UpdateEmailLogStatus(db, log.ID, StatusForwarded, "")
		if err != nil {
			t.Errorf("UpdateEmailLogStatus failed: %v", err)
		}

		// 验证更新
		updated, _ := GetEmailLogByMessageID(db, "msg123")
		if updated.Status != StatusForwarded {
			t.Errorf("Expected status %s, got %s", StatusForwarded, updated.Status)
		}

		if updated.ProcessedAt == nil {
			t.Error("Expected ProcessedAt to be set")
		}
	})

	t.Run("UpdateEmailLogForwarded", func(t *testing.T) {
		log, _ := GetEmailLogByMessageID(db, "msg001")

		err := UpdateEmailLogForwarded(db, log.ID, "forwarded@example.com")
		if err != nil {
			t.Errorf("UpdateEmailLogForwarded failed: %v", err)
		}

		// 验证更新
		updated, _ := GetEmailLogByMessageID(db, "msg001")
		if updated.ForwardedTo != "forwarded@example.com" {
			t.Errorf("Expected ForwardedTo 'forwarded@example.com', got %s", updated.ForwardedTo)
		}

		if updated.Status != StatusForwarded {
			t.Errorf("Expected status %s, got %s", StatusForwarded, updated.Status)
		}
	})
}

func TestEmailLogStatus(t *testing.T) {
	t.Run("StatusConstants", func(t *testing.T) {
		expectedStatuses := map[EmailStatus]string{
			StatusPending:   "pending",
			StatusForwarded: "forwarded",
			StatusFailed:    "failed",
			StatusSkipped:   "skipped",
		}

		for status, expected := range expectedStatuses {
			if string(status) != expected {
				t.Errorf("Expected status %s, got %s", expected, string(status))
			}
		}
	})

	t.Run("TableName", func(t *testing.T) {
		log := EmailLog{}
		tableName := log.TableName()
		if tableName != "email_logs" {
			t.Errorf("Expected table name 'email_logs', got %s", tableName)
		}
	})
}
