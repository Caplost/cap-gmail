package services

import (
	"gmail-forwarding-system/models"
	"strings"

	"gorm.io/gorm"
)

type EmailParser struct {
	db *gorm.DB
}

type ParsedEmailInfo struct {
	Keyword        string
	TargetName     string
	Target         *models.ForwardingTarget
	ShouldForward  bool
	MatchedTargets []models.ForwardingTarget // 支持多目标匹配
}

// NewEmailParser 创建邮件解析器
func NewEmailParser(db *gorm.DB) *EmailParser {
	return &EmailParser{db: db}
}

// ParseEmailSubject 解析邮件主题（支持模糊关键词匹配）
func (ep *EmailParser) ParseEmailSubject(subject string) (*ParsedEmailInfo, error) {
	if subject == "" {
		return &ParsedEmailInfo{ShouldForward: false}, nil
	}

	// 检查是否符合 " - 转发对象名字" 格式
	parts := strings.Split(subject, " - ")
	if len(parts) != 2 {
		return &ParsedEmailInfo{ShouldForward: false}, nil
	}

	content := strings.TrimSpace(parts[0])    // 邮件内容部分
	targetName := strings.TrimSpace(parts[1]) // 转发对象名字

	if content == "" || targetName == "" {
		return &ParsedEmailInfo{ShouldForward: false}, nil
	}

	// 查找指定的转发对象
	target, err := models.GetForwardingTargetByName(ep.db, targetName)
	if err != nil {
		return &ParsedEmailInfo{ShouldForward: false}, nil
	}

	// 策略1: 严格格式匹配（内容部分是单一关键字且精确匹配）
	if ep.isExactKeywordMatch(content, target.Keywords) {
		return &ParsedEmailInfo{
			Keyword:       content,
			TargetName:    targetName,
			Target:        target,
			ShouldForward: true,
		}, nil
	}

	// 策略2: 模糊格式匹配（内容部分包含目标的关键词）
	if matchedKeyword := ep.findKeywordInContent(content, target.Keywords); matchedKeyword != "" {
		return &ParsedEmailInfo{
			Keyword:       matchedKeyword,
			TargetName:    targetName,
			Target:        target,
			ShouldForward: true,
		}, nil
	}

	// 没有匹配到关键词
	return &ParsedEmailInfo{ShouldForward: false}, nil
}

// isExactKeywordMatch 检查是否为精确关键词匹配（严格格式）
func (ep *EmailParser) isExactKeywordMatch(content, keywords string) bool {
	if keywords == "" {
		return true
	}

	keywordList := strings.Split(keywords, ",")
	for _, keyword := range keywordList {
		keyword = strings.TrimSpace(keyword)
		if strings.EqualFold(keyword, content) {
			return true
		}
	}

	return false
}

// findKeywordInContent 在内容中查找关键词（模糊格式）
func (ep *EmailParser) findKeywordInContent(content, keywords string) string {
	if keywords == "" {
		return ""
	}

	// 将内容转为小写进行比较
	contentLower := strings.ToLower(content)

	// 分割关键词列表
	keywordList := strings.Split(keywords, ",")
	for _, keyword := range keywordList {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		keywordLower := strings.ToLower(keyword)

		// 检查内容是否包含该关键词
		if strings.Contains(contentLower, keywordLower) {
			return keyword
		}
	}

	return ""
}

// ValidateEmailFormat 验证邮件主题格式
func (ep *EmailParser) ValidateEmailFormat(subject string) bool {
	return subject != "" && len(strings.TrimSpace(subject)) > 0
}

// GetSupportedFormats 获取支持的主题格式说明
func (ep *EmailParser) GetSupportedFormats() []string {
	return []string{
		"严格格式: '关键字 - 转发对象名字' (关键字精确匹配目标的关键词列表)",
		"模糊格式: '邮件内容 - 转发对象名字' (内容包含目标的关键词即可)",
		"优先级: 严格格式 > 模糊格式",
		"示例: '工作 - 工作邮件' (严格) 或 '工作安排通知 - 张三' (模糊)",
	}
}
