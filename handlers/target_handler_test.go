package handlers

import (
	"bytes"
	"encoding/json"
	"gmail-forwarding-system/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTargetHandlerTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	db.AutoMigrate(&models.ForwardingTarget{})
	return db
}

func setupTargetRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handler := NewTargetHandler(db)
	api := r.Group("/api")
	targets := api.Group("/targets")
	{
		targets.GET("", handler.GetTargets)
		targets.GET("/:id", handler.GetTarget)
		targets.POST("", handler.CreateTarget)
		targets.PUT("/:id", handler.UpdateTarget)
		targets.DELETE("/:id", handler.DeleteTarget)
	}

	return r
}

func TestTargetHandler(t *testing.T) {
	db := setupTargetHandlerTestDB()
	router := setupTargetRouter(db)

	t.Run("CreateTarget", func(t *testing.T) {
		target := CreateTargetRequest{
			Name:     "测试用户",
			Email:    "test@example.com",
			Keywords: "客服,售后",
		}

		jsonData, _ := json.Marshal(target)
		req, _ := http.NewRequest("POST", "/api/targets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["message"] != "转发对象创建成功" {
			t.Errorf("Unexpected response message: %v", response["message"])
		}
	})

	t.Run("GetTargets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/targets", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		data, exists := response["data"]
		if !exists {
			t.Error("Expected 'data' field in response")
		}

		targets, ok := data.([]interface{})
		if !ok {
			t.Error("Expected 'data' to be an array")
		}

		if len(targets) == 0 {
			t.Error("Expected at least one target")
		}
	})

	t.Run("GetTargetByID", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/targets/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("UpdateTarget", func(t *testing.T) {
		updateData := UpdateTargetRequest{
			Name:     "更新用户",
			Keywords: "客服,售后,投诉",
		}

		jsonData, _ := json.Marshal(updateData)
		req, _ := http.NewRequest("PUT", "/api/targets/1", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("DeleteTarget", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/targets/1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

func TestTargetHandlerValidation(t *testing.T) {
	db := setupTargetHandlerTestDB()
	router := setupTargetRouter(db)

	t.Run("CreateTargetWithInvalidEmail", func(t *testing.T) {
		target := CreateTargetRequest{
			Name:  "测试用户",
			Email: "invalid-email",
		}

		jsonData, _ := json.Marshal(target)
		req, _ := http.NewRequest("POST", "/api/targets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid email, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("CreateTargetWithMissingFields", func(t *testing.T) {
		target := CreateTargetRequest{
			Email: "test@example.com",
			// Missing Name field
		}

		jsonData, _ := json.Marshal(target)
		req, _ := http.NewRequest("POST", "/api/targets", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for missing fields, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("GetNonExistentTarget", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/targets/999", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status %d for non-existent target, got %d", http.StatusNotFound, w.Code)
		}
	})

	t.Run("InvalidIDFormat", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/targets/invalid", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid ID format, got %d", http.StatusBadRequest, w.Code)
		}
	})
}
