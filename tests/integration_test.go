package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gmail-forwarding-system/handlers"
	"gmail-forwarding-system/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupIntegrationTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	// 自动迁移所有表
	db.AutoMigrate(&models.ForwardingTarget{}, &models.EmailLog{})

	return db
}

func setupIntegrationRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// 设置路由
	targetHandler := handlers.NewTargetHandler(db)

	api := r.Group("/api")
	targets := api.Group("/targets")
	{
		targets.GET("", targetHandler.GetTargets)
		targets.GET("/:id", targetHandler.GetTarget)
		targets.POST("", targetHandler.CreateTarget)
		targets.PUT("/:id", targetHandler.UpdateTarget)
		targets.DELETE("/:id", targetHandler.DeleteTarget)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Gmail转发系统运行正常",
		})
	})

	return r
}

func TestIntegrationWorkflow(t *testing.T) {
	db := setupIntegrationTestDB()
	router := setupIntegrationRouter(db)

	t.Run("CompleteTargetManagementWorkflow", func(t *testing.T) {
		// 1. 创建转发对象
		createTarget := handlers.CreateTargetRequest{
			Name:     "张三",
			Email:    "zhangsan@example.com",
			Keywords: "客服,售后",
		}

		jsonData, _ := json.Marshal(createTarget)
		req, _ := http.NewRequest("POST", "/api/targets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create target: status %d", w.Code)
		}

		// 解析创建响应
		var createResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &createResponse)
		data := createResponse["data"].(map[string]interface{})
		targetID := int(data["id"].(float64))

		// 2. 获取所有转发对象
		req, _ = http.NewRequest("GET", "/api/targets", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to get targets: status %d", w.Code)
		}

		var getResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &getResponse)
		targets := getResponse["data"].([]interface{})

		if len(targets) == 0 {
			t.Fatal("Expected at least one target")
		}

		// 3. 获取单个转发对象
		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/targets/%d", targetID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to get single target: status %d", w.Code)
		}

		// 4. 更新转发对象
		updateTarget := handlers.UpdateTargetRequest{
			Name:     "张三（更新）",
			Keywords: "客服,售后,投诉",
		}

		jsonData, _ = json.Marshal(updateTarget)
		req, _ = http.NewRequest("PUT", fmt.Sprintf("/api/targets/%d", targetID), bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to update target: status %d", w.Code)
		}

		// 5. 验证更新
		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/targets/%d", targetID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var updateResponse map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &updateResponse)
		updatedData := updateResponse["data"].(map[string]interface{})

		if updatedData["name"].(string) != "张三（更新）" {
			t.Errorf("Expected updated name '张三（更新）', got %s", updatedData["name"])
		}

		// 6. 删除转发对象
		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/targets/%d", targetID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Failed to delete target: status %d", w.Code)
		}

		// 7. 验证删除（软删除）
		req, _ = http.NewRequest("GET", "/api/targets", nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		json.Unmarshal(w.Body.Bytes(), &getResponse)
		remainingTargets := getResponse["data"].([]interface{})

		if len(remainingTargets) != 0 {
			t.Error("Expected no active targets after deletion")
		}
	})

	t.Run("HealthCheck", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check failed: status %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", response["status"])
		}
	})
}

func TestDatabaseIntegration(t *testing.T) {
	db := setupIntegrationTestDB()

	t.Run("DatabaseOperations", func(t *testing.T) {
		// 测试模型操作
		target := &models.ForwardingTarget{
			Name:     "数据库测试用户",
			Email:    "dbtest@example.com",
			Keywords: "测试",
			IsActive: true,
		}

		// 创建
		err := models.CreateForwardingTarget(db, target)
		if err != nil {
			t.Fatalf("Failed to create target: %v", err)
		}

		// 查询
		targets, err := models.GetForwardingTargets(db)
		if err != nil {
			t.Fatalf("Failed to get targets: %v", err)
		}

		if len(targets) == 0 {
			t.Fatal("Expected at least one target")
		}

		// 按名字查询
		foundTarget, err := models.GetForwardingTargetByName(db, "数据库测试用户")
		if err != nil {
			t.Fatalf("Failed to get target by name: %v", err)
		}

		if foundTarget.Email != "dbtest@example.com" {
			t.Errorf("Expected email 'dbtest@example.com', got %s", foundTarget.Email)
		}

		// 更新
		foundTarget.Keywords = "测试,更新"
		err = models.UpdateForwardingTarget(db, foundTarget)
		if err != nil {
			t.Fatalf("Failed to update target: %v", err)
		}

		// 软删除
		err = models.DeleteForwardingTarget(db, foundTarget.ID)
		if err != nil {
			t.Fatalf("Failed to delete target: %v", err)
		}

		// 验证软删除
		_, err = models.GetForwardingTargetByName(db, "数据库测试用户")
		if err == nil {
			t.Error("Expected error after soft delete")
		}
	})

	t.Run("EmailLogOperations", func(t *testing.T) {
		log := &models.EmailLog{
			MessageID:  "integration-test-001",
			Subject:    "客服 - 集成测试",
			FromEmail:  "sender@test.com",
			ToEmail:    "receiver@test.com",
			Keyword:    "客服",
			TargetName: "集成测试",
			Status:     models.StatusPending,
		}

		// 创建日志
		err := models.CreateEmailLog(db, log)
		if err != nil {
			t.Fatalf("Failed to create email log: %v", err)
		}

		// 查询日志
		logs, total, err := models.GetEmailLogs(db, 1, 10)
		if err != nil {
			t.Fatalf("Failed to get email logs: %v", err)
		}

		if total == 0 {
			t.Fatal("Expected at least one email log")
		}

		if len(logs) == 0 {
			t.Fatal("Expected logs in result")
		}

		// 按MessageID查询
		foundLog, err := models.GetEmailLogByMessageID(db, "integration-test-001")
		if err != nil {
			t.Fatalf("Failed to get log by message ID: %v", err)
		}

		if foundLog.Subject != "客服 - 集成测试" {
			t.Errorf("Expected subject '客服 - 集成测试', got %s", foundLog.Subject)
		}

		// 更新状态
		err = models.UpdateEmailLogStatus(db, foundLog.ID, models.StatusForwarded, "")
		if err != nil {
			t.Fatalf("Failed to update log status: %v", err)
		}

		// 验证状态更新
		updatedLog, _ := models.GetEmailLogByMessageID(db, "integration-test-001")
		if updatedLog.Status != models.StatusForwarded {
			t.Errorf("Expected status %s, got %s", models.StatusForwarded, updatedLog.Status)
		}
	})
}
