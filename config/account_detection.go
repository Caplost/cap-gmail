package config

import (
	"os"
	"strings"
)

// AccountType 账户类型枚举
type AccountType string

const (
	AccountTypeFree      AccountType = "free"
	AccountTypeWorkspace AccountType = "workspace"
	AccountTypeBusiness  AccountType = "business"
)

// AccountLimits 账户发送限制配置
type AccountLimits struct {
	Type            AccountType `json:"type"`
	DailyLimit      int         `json:"daily_limit"`      // 日发送限制
	HourlyLimit     int         `json:"hourly_limit"`     // 小时发送限制
	MinuteLimit     int         `json:"minute_limit"`     // 分钟发送限制
	ConcurrentLimit int         `json:"concurrent_limit"` // 并发连接限制
	MonthlyLimit    int         `json:"monthly_limit"`    // 月发送限制
	Description     string      `json:"description"`      // 账户描述
	RecommendedUse  string      `json:"recommended_use"`  // 推荐使用场景
}

// GetAccountLimits 获取指定账户类型的发送限制
func GetAccountLimits(accountType AccountType) AccountLimits {
	switch accountType {
	case AccountTypeWorkspace:
		return AccountLimits{
			Type:            AccountTypeWorkspace,
			DailyLimit:      2000,
			HourlyLimit:     300,
			MinuteLimit:     30,
			ConcurrentLimit: 10,
			MonthlyLimit:    60000,
			Description:     "Google Workspace (付费企业账户)",
			RecommendedUse:  "中大型企业邮件转发，支持部门级邮件管理",
		}
	case AccountTypeBusiness:
		return AccountLimits{
			Type:            AccountTypeBusiness,
			DailyLimit:      3000,
			HourlyLimit:     500,
			MinuteLimit:     50,
			ConcurrentLimit: 15,
			MonthlyLimit:    90000,
			Description:     "Gmail Business (高级企业账户)",
			RecommendedUse:  "大规模企业邮件转发，支持客服中心等高频场景",
		}
	default: // AccountTypeFree
		return AccountLimits{
			Type:            AccountTypeFree,
			DailyLimit:      500,
			HourlyLimit:     100,
			MinuteLimit:     10,
			ConcurrentLimit: 5,
			MonthlyLimit:    15000,
			Description:     "Gmail 免费账户",
			RecommendedUse:  "个人或小型团队邮件转发，轻量级使用",
		}
	}
}

// DetectAccountType 检测Gmail账户类型
func DetectAccountType(email string) AccountType {
	if email == "" {
		return AccountTypeFree
	}

	email = strings.ToLower(strings.TrimSpace(email))

	// 手动指定账户类型（环境变量优先）
	if envType := os.Getenv("GMAIL_ACCOUNT_TYPE"); envType != "" {
		switch strings.ToLower(envType) {
		case "workspace", "gsuite":
			return AccountTypeWorkspace
		case "business", "enterprise":
			return AccountTypeBusiness
		case "free", "personal":
			return AccountTypeFree
		}
	}

	if !strings.Contains(email, "@") {
		return AccountTypeFree
	}

	domain := strings.Split(email, "@")[1]

	// Google Workspace检测：非gmail.com域名通常是企业账户
	if domain != "gmail.com" && domain != "googlemail.com" {
		return AccountTypeWorkspace
	}

	// Gmail Business检测标识符
	businessIdentifiers := []string{
		"+business", ".business", "+enterprise", ".enterprise",
		"+company", ".company", "+corp", ".corp",
	}

	for _, identifier := range businessIdentifiers {
		if strings.Contains(email, identifier) {
			return AccountTypeBusiness
		}
	}

	return AccountTypeFree
}

// UpgradeRecommendations 获取账户升级建议
func UpgradeRecommendations(currentType AccountType, currentUsage int) map[string]interface{} {
	current := GetAccountLimits(currentType)

	recommendations := map[string]interface{}{
		"current_account":  current,
		"current_usage":    currentUsage,
		"usage_percentage": float64(currentUsage) / float64(current.DailyLimit) * 100,
	}

	// 如果使用率超过80%，建议升级
	if float64(currentUsage)/float64(current.DailyLimit) > 0.8 {
		switch currentType {
		case AccountTypeFree:
			workspace := GetAccountLimits(AccountTypeWorkspace)
			business := GetAccountLimits(AccountTypeBusiness)
			recommendations["upgrade_needed"] = true
			recommendations["workspace_option"] = workspace
			recommendations["business_option"] = business
			recommendations["cost_benefit"] = map[string]interface{}{
				"workspace_increase": workspace.DailyLimit - current.DailyLimit,
				"business_increase":  business.DailyLimit - current.DailyLimit,
				"workspace_ratio":    float64(workspace.DailyLimit) / float64(current.DailyLimit),
				"business_ratio":     float64(business.DailyLimit) / float64(current.DailyLimit),
			}
		case AccountTypeWorkspace:
			business := GetAccountLimits(AccountTypeBusiness)
			recommendations["upgrade_needed"] = true
			recommendations["business_option"] = business
			recommendations["cost_benefit"] = map[string]interface{}{
				"business_increase": business.DailyLimit - current.DailyLimit,
				"business_ratio":    float64(business.DailyLimit) / float64(current.DailyLimit),
			}
		default:
			recommendations["upgrade_needed"] = false
			recommendations["message"] = "您已使用最高级别账户"
		}
	} else {
		recommendations["upgrade_needed"] = false
		recommendations["message"] = "当前账户配额充足"
	}

	return recommendations
}

// GetOptimalSendingStrategy 获取最优发送策略
func GetOptimalSendingStrategy(accountType AccountType) map[string]interface{} {
	limits := GetAccountLimits(accountType)

	strategy := map[string]interface{}{
		"account_limits": limits,
		"smtp_advantage": true,
		"api_comparison": map[string]interface{}{
			"gmail_api_limit":    "250配额/秒/用户",
			"smtp_daily_limit":   limits.DailyLimit,
			"smtp_no_sec_limit":  true,
			"smtp_advantage_msg": "SMTP无秒级限制，适合高频发送",
		},
	}

	// 根据账户类型给出具体建议
	switch accountType {
	case AccountTypeFree:
		strategy["recommendations"] = []string{
			"使用SMTP突破Gmail API秒级限制",
			"合理控制发送频率，避免超出日限制",
			"考虑升级到付费账户获得更高配额",
			"重要邮件使用Gmail API确保送达率",
		}
	case AccountTypeWorkspace:
		strategy["recommendations"] = []string{
			"充分利用2000封/天的SMTP配额",
			"可支持中等规模的邮件转发系统",
			"混合使用SMTP和Gmail API获得最佳性能",
			"设置合理的发送间隔避免被识别为垃圾邮件",
		}
	case AccountTypeBusiness:
		strategy["recommendations"] = []string{
			"享受3000封/天的企业级配额",
			"可构建大规模邮件转发系统",
			"优先使用SMTP获得最高发送效率",
			"配置监控系统跟踪发送状态和配额使用",
		}
	}

	return strategy
}

// ValidateAccountConfiguration 验证账户配置
func ValidateAccountConfiguration() (bool, []string) {
	var issues []string

	// 检查SMTP配置
	if AppConfig.SMTPUser == "" {
		issues = append(issues, "SMTP_USER未配置")
	}
	if AppConfig.SMTPPassword == "" {
		issues = append(issues, "SMTP_PASSWORD未配置")
	}

	// 检查账户类型检测
	detectedType := DetectAccountType(AppConfig.SMTPUser)
	if detectedType == AccountTypeFree && AppConfig.SMTPUser != "" {
		if !strings.HasSuffix(AppConfig.SMTPUser, "@gmail.com") {
			issues = append(issues, "账户类型检测可能不准确，建议设置GMAIL_ACCOUNT_TYPE环境变量")
		}
	}

	return len(issues) == 0, issues
}
