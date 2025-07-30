package utils

import (
	"testing"
)

func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "user@example.com",
			expected:    "us*r@example.com",
			description: "标准邮箱脱敏",
		},
		{
			input:       "test@gmail.com",
			expected:    "te*t@gmail.com",
			description: "短用户名邮箱脱敏",
		},
		{
			input:       "verylongusername@domain.com",
			expected:    "ve*************e@domain.com",
			description: "长用户名邮箱脱敏",
		},
		{
			input:       "a@example.com",
			expected:    "a***@example.com",
			description: "单字符用户名",
		},
		{
			input:       "ab@example.com",
			expected:    "a***@example.com",
			description: "双字符用户名",
		},
		{
			input:       "abc@example.com",
			expected:    "a***@example.com",
			description: "三字符用户名",
		},
		{
			input:       "",
			expected:    "",
			description: "空字符串",
		},
		{
			input:       "notanemail",
			expected:    "notanemail",
			description: "无效邮箱格式",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := MaskEmail(tc.input)
			if result != tc.expected {
				t.Errorf("MaskEmail('%s'): expected '%s', got '%s'",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestMaskSubject(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "客服问题需要处理",
			expected:    "客服问题****",
			description: "标准主题脱敏",
		},
		{
			input:       "这是一个很长的邮件主题用于测试脱敏功能",
			expected:    "这是一个很*********试脱敏功能",
			description: "长主题脱敏",
		},
		{
			input:       "短主题",
			expected:    "短**",
			description: "短主题脱敏",
		},
		{
			input:       "",
			expected:    "",
			description: "空主题",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := MaskSubject(tc.input)
			if result != tc.expected {
				t.Errorf("MaskSubject('%s'): expected '%s', got '%s'",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email       string
		isValid     bool
		description string
	}{
		{"user@example.com", true, "标准邮箱"},
		{"test.email@domain.co.uk", true, "带点的用户名和多级域名"},
		{"user+tag@example.com", true, "带加号的邮箱"},
		{"123@example.com", true, "数字用户名"},
		{"user@123.com", true, "数字域名"},
		{"", false, "空邮箱"},
		{"notanemail", false, "无@符号"},
		{"@example.com", false, "缺少用户名"},
		{"user@", false, "缺少域名"},
		{"user@.com", false, "域名格式错误"},
		{"user@example.", false, "缺少顶级域名"},
		{"user name@example.com", false, "用户名包含空格"},
		{"user@exam ple.com", false, "域名包含空格"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := IsValidEmail(tc.email)
			if result != tc.isValid {
				t.Errorf("IsValidEmail('%s'): expected %v, got %v",
					tc.email, tc.isValid, result)
			}
		})
	}
}

func TestGenerateHash(t *testing.T) {
	t.Run("ConsistentHashing", func(t *testing.T) {
		input := "test input"
		hash1 := GenerateHash(input)
		hash2 := GenerateHash(input)

		if hash1 != hash2 {
			t.Errorf("Hash should be consistent: %s != %s", hash1, hash2)
		}
	})

	t.Run("DifferentInputsDifferentHashes", func(t *testing.T) {
		hash1 := GenerateHash("input1")
		hash2 := GenerateHash("input2")

		if hash1 == hash2 {
			t.Error("Different inputs should produce different hashes")
		}
	})

	t.Run("HashLength", func(t *testing.T) {
		hash := GenerateHash("test")
		if len(hash) != 32 { // MD5 hash is 32 characters
			t.Errorf("Expected hash length 32, got %d", len(hash))
		}
	})
}

func TestSanitizeInput(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		description string
	}{
		{
			input:       "normal input",
			expected:    "normal input",
			description: "正常输入",
		},
		{
			input:       "<script>alert(xss)</script>",
			expected:    "scriptalertxss/script",
			description: "移除HTML标签",
		},
		{
			input:       "SELECT * FROM users; DROP TABLE users;",
			expected:    "SELECT * FROM users DROP TABLE users",
			description: "移除SQL注入字符",
		},
		{
			input:       "  spaces around  ",
			expected:    "spaces around",
			description: "修剪空格",
		},
		{
			input:       "",
			expected:    "",
			description: "空输入",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := SanitizeInput(tc.input)
			if result != tc.expected {
				t.Errorf("SanitizeInput('%s'): expected '%s', got '%s'",
					tc.input, tc.expected, result)
			}
		})
	}
}

func TestIsValidKeyword(t *testing.T) {
	testCases := []struct {
		keyword     string
		isValid     bool
		description string
	}{
		{"客服", true, "中文关键字"},
		{"support", true, "英文关键字"},
		{"客服123", true, "中英文数字混合"},
		{"customer-service", true, "带连字符"},
		{"client_support", true, "带下划线"},
		{"", false, "空关键字"},
		{"a", true, "单字符"},
		{string(make([]byte, 51)), false, "超长关键字"}, // 超过50字符
		{"客服问题（紧急）", true, "带括号"},
		{"normal keyword", true, "带空格"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			result := IsValidKeyword(tc.keyword)
			if result != tc.isValid {
				t.Errorf("IsValidKeyword('%s'): expected %v, got %v",
					tc.keyword, tc.isValid, result)
			}
		})
	}
}
