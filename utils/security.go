package utils

import (
	"crypto/md5"
	"fmt"
	"regexp"
	"strings"
)

// MaskEmail 脱敏邮箱地址
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	// 找到@符号位置
	atIndex := strings.Index(email, "@")
	if atIndex == -1 {
		return email // 不是有效邮箱格式
	}

	username := email[:atIndex]
	domain := email[atIndex:]

	// 用户名脱敏：显示前2位和后1位，中间用*代替
	if len(username) <= 3 {
		return email[:1] + "***" + domain
	}

	maskedUsername := username[:2] + strings.Repeat("*", len(username)-3) + username[len(username)-1:]
	return maskedUsername + domain
}

// MaskSubject 脱敏邮件主题 (使用rune处理中文字符)
func MaskSubject(subject string) string {
	runes := []rune(subject)
	if len(runes) <= 10 {
		half := len(runes) / 2
		return string(runes[:half]) + strings.Repeat("*", len(runes)-half)
	}

	return string(runes[:5]) + strings.Repeat("*", len(runes)-10) + string(runes[len(runes)-5:])
}

// IsValidEmail 验证邮箱格式
func IsValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(email)
}

// GenerateHash 生成字符串哈希值
func GenerateHash(input string) string {
	hash := md5.Sum([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// SanitizeInput 清理输入字符串，防止注入
func SanitizeInput(input string) string {
	// 移除潜在危险字符
	dangerous := []string{"<", ">", "'", "\"", "&", ";", "(", ")", "{", "}", "[", "]"}
	result := input

	for _, char := range dangerous {
		result = strings.ReplaceAll(result, char, "")
	}

	return strings.TrimSpace(result)
}

// IsValidKeyword 验证关键字格式
func IsValidKeyword(keyword string) bool {
	if len(keyword) == 0 || len(keyword) > 50 {
		return false
	}

	// 只允许字母、数字、中文、空格和常见标点
	pattern := `^[\p{L}\p{N}\s\-_.,!?()（）]+$`
	reg := regexp.MustCompile(pattern)
	return reg.MatchString(keyword)
}

// SecurityHeaders 安全响应头
var SecurityHeaders = map[string]string{
	"X-Content-Type-Options":    "nosniff",
	"X-Frame-Options":           "DENY",
	"X-XSS-Protection":          "1; mode=block",
	"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	"Content-Security-Policy":   "default-src 'self'",
	"Referrer-Policy":           "strict-origin-when-cross-origin",
}
