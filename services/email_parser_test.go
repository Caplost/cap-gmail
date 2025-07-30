package services

import (
	"gmail-forwarding-system/models"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupParserTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("failed to connect test database")
	}

	db.AutoMigrate(&models.ForwardingTarget{})

	// 插入测试数据
	targets := []models.ForwardingTarget{
		{Name: "张三", Email: "zhangsan@example.com", Keywords: "客服,售后", IsActive: true},
		{Name: "李四", Email: "lisi@example.com", Keywords: "技术支持", IsActive: true},
		{Name: "王五", Email: "wangwu@example.com", Keywords: "", IsActive: true}, // 无关键字限制
	}

	for _, target := range targets {
		db.Create(&target)
	}

	// 创建未启用的用户 - 使用两步法避免GORM默认值问题
	inactiveTarget := models.ForwardingTarget{
		Name:     "赵六",
		Email:    "zhaoliu@example.com",
		Keywords: "投诉",
		IsActive: true,
	}
	db.Create(&inactiveTarget)
	db.Model(&inactiveTarget).Update("is_active", false)

	return db
}

func TestEmailParser(t *testing.T) {
	db := setupParserTestDB()
	parser := NewEmailParser(db)

	t.Run("ValidEmailSubjectParsing", func(t *testing.T) {
		testCases := []struct {
			subject         string
			expectedKeyword string
			expectedTarget  string
			shouldForward   bool
			description     string
		}{
			{
				subject:         "客服 - 张三",
				expectedKeyword: "客服",
				expectedTarget:  "张三",
				shouldForward:   true,
				description:     "标准格式，匹配关键字",
			},
			{
				subject:         "技术支持 - 李四",
				expectedKeyword: "技术支持",
				expectedTarget:  "李四",
				shouldForward:   true,
				description:     "匹配完整关键字",
			},
			{
				subject:         "任何关键字 - 王五",
				expectedKeyword: "任何关键字",
				expectedTarget:  "王五",
				shouldForward:   true,
				description:     "无关键字限制的转发对象",
			},
			{
				subject:         "投诉 - 赵六",
				expectedKeyword: "投诉",
				expectedTarget:  "赵六",
				shouldForward:   false,
				description:     "转发对象未启用",
			},
			{
				subject:         "其他问题 - 张三",
				expectedKeyword: "其他问题",
				expectedTarget:  "张三",
				shouldForward:   false,
				description:     "关键字不匹配",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				result, err := parser.ParseEmailSubject(tc.subject)
				if err != nil {
					t.Errorf("ParseEmailSubject failed: %v", err)
				}

				if result.Keyword != tc.expectedKeyword {
					t.Errorf("Expected keyword '%s', got '%s'", tc.expectedKeyword, result.Keyword)
				}

				if result.TargetName != tc.expectedTarget {
					t.Errorf("Expected target '%s', got '%s'", tc.expectedTarget, result.TargetName)
				}

				if result.ShouldForward != tc.shouldForward {
					t.Errorf("Expected ShouldForward %v, got %v", tc.shouldForward, result.ShouldForward)
				}
			})
		}
	})

	t.Run("InvalidEmailSubjectFormats", func(t *testing.T) {
		invalidSubjects := []string{
			"",             // 空主题
			"客服张三",         // 缺少分隔符
			"客服 - ",        // 缺少转发对象
			" - 张三",        // 缺少关键字
			"客服-张三",        // 错误的分隔符
			"客服 -- 张三",     // 多个分隔符
			"客服 - 张三 - 其他", // 多段格式
		}

		for _, subject := range invalidSubjects {
			t.Run("Invalid: "+subject, func(t *testing.T) {
				result, err := parser.ParseEmailSubject(subject)
				if err != nil {
					t.Errorf("ParseEmailSubject should not error on invalid format: %v", err)
				}

				if result.ShouldForward {
					t.Errorf("Invalid format should not be forwarded: '%s'", subject)
				}
			})
		}
	})

	t.Run("NonExistentTarget", func(t *testing.T) {
		result, err := parser.ParseEmailSubject("客服 - 不存在的用户")
		if err != nil {
			t.Errorf("ParseEmailSubject failed: %v", err)
		}

		if result.ShouldForward {
			t.Error("Non-existent target should not be forwarded")
		}

		if result.TargetName != "不存在的用户" {
			t.Errorf("Expected target name '不存在的用户', got '%s'", result.TargetName)
		}
	})
}

func TestKeywordMatching(t *testing.T) {
	db := setupParserTestDB()
	parser := NewEmailParser(db)

	t.Run("CaseInsensitiveMatching", func(t *testing.T) {
		// 测试大小写不敏感匹配
		testCases := []string{
			"客服 - 张三",
			"KEFU - 张三", // 需要在数据库中添加相应的测试数据
		}

		for _, subject := range testCases {
			result, err := parser.ParseEmailSubject(subject)
			if err != nil {
				t.Errorf("ParseEmailSubject failed: %v", err)
			}

			// 第一个测试用例应该匹配
			if subject == "客服 - 张三" && !result.ShouldForward {
				t.Errorf("Expected to forward '%s'", subject)
			}
		}
	})

	t.Run("MultipleKeywords", func(t *testing.T) {
		// 张三的关键字: "客服,售后"
		testCases := []struct {
			subject       string
			shouldForward bool
		}{
			{"客服 - 张三", true},  // 匹配第一个关键字
			{"售后 - 张三", true},  // 匹配第二个关键字
			{"技术 - 张三", false}, // 不匹配任何关键字
		}

		for _, tc := range testCases {
			result, err := parser.ParseEmailSubject(tc.subject)
			if err != nil {
				t.Errorf("ParseEmailSubject failed: %v", err)
			}

			if result.ShouldForward != tc.shouldForward {
				t.Errorf("Subject '%s': expected ShouldForward %v, got %v",
					tc.subject, tc.shouldForward, result.ShouldForward)
			}
		}
	})
}

func TestEmailFormatValidation(t *testing.T) {
	db := setupParserTestDB()
	parser := NewEmailParser(db)

	t.Run("ValidateEmailFormat", func(t *testing.T) {
		testCases := []struct {
			subject string
			isValid bool
		}{
			{"客服 - 张三", true},
			{"技术支持 - 李四", true},
			{"", false},
			{"客服张三", false},
			{"客服 - ", false},
			{" - 张三", false},
		}

		for _, tc := range testCases {
			isValid := parser.ValidateEmailFormat(tc.subject)
			if isValid != tc.isValid {
				t.Errorf("ValidateEmailFormat('%s'): expected %v, got %v",
					tc.subject, tc.isValid, isValid)
			}
		}
	})
}
