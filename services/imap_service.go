package services

import (
	"fmt"
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/utils"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// IMAPService IMAP邮件接收服务
type IMAPService struct {
	host     string
	port     int
	username string
	password string // 应用密码
}

// NewIMAPService 创建IMAP服务实例
func NewIMAPService() *IMAPService {
	cfg := config.AppConfig

	// 检查IMAP配置
	if cfg.IMAPHost == "" || cfg.IMAPUser == "" || cfg.IMAPPassword == "" {
		utils.LogWarn("IMAP配置不完整，跳过IMAP服务初始化")
		return nil
	}

	service := &IMAPService{
		host:     cfg.IMAPHost,
		port:     cfg.IMAPPort,
		username: cfg.IMAPUser,
		password: cfg.IMAPPassword,
	}

	// 测试连接
	if err := service.TestConnection(); err != nil {
		utils.LogError("IMAP连接测试失败", err)
		return nil
	}

	utils.LogInfo("IMAP服务初始化成功")
	return service
}

// GetUnreadEmails 获取未读邮件
func (is *IMAPService) GetUnreadEmails() ([]*EmailData, error) {
	// 连接IMAP服务器
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", is.host, is.port), nil)
	if err != nil {
		return nil, fmt.Errorf("连接IMAP服务器失败: %v", err)
	}
	defer c.Logout()

	// 登录
	if err := c.Login(is.username, is.password); err != nil {
		return nil, fmt.Errorf("IMAP登录失败: %v", err)
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("选择收件箱失败: %v", err)
	}

	// 搜索未读邮件
	criteria := imap.NewSearchCriteria()
	criteria.WithoutFlags = []string{imap.SeenFlag}

	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("搜索未读邮件失败: %v", err)
	}

	if len(uids) == 0 {
		utils.LogInfo("📭 暂无新邮件")
		return []*EmailData{}, nil
	}

	utils.LogInfo(fmt.Sprintf("找到 %d 封未读邮件", len(uids)))

	// 创建序列集
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	// 获取邮件，包含完整邮件体
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// 获取邮件的envelope、body和完整内容
	fetchItems := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchRFC822, // 获取完整的原始邮件
	}

	go func() {
		done <- c.Fetch(seqset, fetchItems, messages)
	}()

	var emails []*EmailData
	for msg := range messages {
		email, err := is.parseMessage(msg)
		if err != nil {
			utils.LogWarn(fmt.Sprintf("解析邮件失败: %v", err))
			continue
		}
		emails = append(emails, email)
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("获取邮件内容失败: %v", err)
	}

	return emails, nil
}

// parseMessage 解析IMAP邮件消息
func (is *IMAPService) parseMessage(msg *imap.Message) (*EmailData, error) {
	if msg.Envelope == nil {
		return nil, fmt.Errorf("邮件信封为空")
	}

	// 解析发件人
	var fromEmail string
	if len(msg.Envelope.From) > 0 {
		fromEmail = msg.Envelope.From[0].Address()
	}

	// 解析主题
	subject := msg.Envelope.Subject

	// 获取MessageID
	messageID := msg.Envelope.MessageId
	if messageID == "" {
		// 如果没有MessageID，生成一个基于时间和序号的唯一标识
		messageID = fmt.Sprintf("imap-%d-%d@local", msg.SeqNum, time.Now().Unix())
	}

	// 解析日期
	receivedTime := msg.Envelope.Date
	if receivedTime.IsZero() {
		receivedTime = time.Now()
	}

	// 直接获取原始邮件内容，不做复杂解析
	body := is.getRawEmailBody(msg)

	emailData := &EmailData{
		ID:           strconv.Itoa(int(msg.SeqNum)),
		MessageID:    messageID,
		From:         fromEmail,
		Subject:      subject,
		Body:         body,
		ReceivedTime: receivedTime,
	}

	return emailData, nil
}

// getRawEmailBody 直接获取原始邮件内容（修复重复问题）
func (is *IMAPService) getRawEmailBody(msg *imap.Message) string {
	// 优先使用RFC822完整邮件内容，只取第一个有效内容避免重复
	for section, reader := range msg.Body {
		if reader != nil {
			bodyBytes, err := io.ReadAll(reader)
			if err == nil && len(bodyBytes) > 0 {
				rawContent := string(bodyBytes)

				// 如果看起来像完整邮件（包含邮件头），提取正文部分
				if strings.Contains(rawContent, "Subject:") || strings.Contains(rawContent, "From:") {
					if body := is.extractBodyFromRFC822(rawContent); body != "" {
						fmt.Printf("🔧 IMAP获取RFC822格式内容，长度: %d\n", len(body))
						return body // 立即返回，不处理其他section
					}
				} else {
					// 否则直接使用原始内容（不在这里解码，留给发送前处理）
					cleaned := is.simpleCleanup(rawContent)
					if strings.TrimSpace(cleaned) != "" {
						fmt.Printf("🔧 IMAP获取原始内容，长度: %d\n", len(cleaned))
						return cleaned // 立即返回，不处理其他section
					}
				}
			}
		}

		// 移除调试信息
		_ = section
	}

	// 如果没有内容，返回基本信息
	return fmt.Sprintf("邮件主题: %s\n[收到邮件，但正文内容为空]", msg.Envelope.Subject)
}

// extractBodyFromRFC822 从RFC822格式邮件中提取正文（修复重复问题）
func (is *IMAPService) extractBodyFromRFC822(rawEmail string) string {
	lines := strings.Split(rawEmail, "\n")
	var bodyLines []string
	var inBody bool
	var inMimeSection bool
	var foundFirstTextPart bool

	for _, line := range lines {
		line = strings.TrimRight(line, "\r")

		// 邮件头结束标志（空行）
		if !inBody && strings.TrimSpace(line) == "" {
			inBody = true
			continue
		}

		// 如果已经进入正文区域
		if inBody {
			// 检测MIME边界开始
			if strings.HasPrefix(line, "--") && len(line) > 10 {
				// 如果已经找到第一个文本部分，停止处理
				if foundFirstTextPart {
					fmt.Printf("🔧 检测到MIME边界，已找到第一个文本部分，停止提取\n")
					break
				}
				inMimeSection = false
				continue
			}

			// 检测Content-Type
			if strings.HasPrefix(strings.ToLower(line), "content-type:") {
				lowerLine := strings.ToLower(line)
				if strings.Contains(lowerLine, "text/plain") || strings.Contains(lowerLine, "text/html") {
					inMimeSection = true
					displayLine := line
					if len(line) > 50 {
						displayLine = line[:50] + "..."
					}
					fmt.Printf("🔧 找到文本内容部分: %s\n", displayLine)
				} else {
					inMimeSection = false // 跳过非文本部分
				}
				continue
			}

			// 跳过其他Content-*头部
			if strings.HasPrefix(line, "Content-") {
				continue
			}

			// 如果在文本MIME段中，收集内容
			if inMimeSection || !strings.Contains(rawEmail, "Content-Type:") {
				// 跳过空行开头
				if len(bodyLines) == 0 && strings.TrimSpace(line) == "" {
					continue
				}

				if strings.TrimSpace(line) != "" || len(bodyLines) > 0 {
					bodyLines = append(bodyLines, line)
					foundFirstTextPart = true
				}
			}
		}
	}

	if len(bodyLines) == 0 {
		return ""
	}

	body := strings.Join(bodyLines, "\n")
	fmt.Printf("🔧 提取的原始正文长度: %d\n", len(body))

	// 不在这里解码，保持原始状态给发送前处理
	return strings.TrimSpace(body)
}

// simpleCleanup 简单清理邮件内容（期望接收已解码的内容）
func (is *IMAPService) simpleCleanup(content string) string {
	// 移除过长的分隔线
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过过长的重复字符行（通常是分隔线）
		if len(line) > 50 && is.isRepeatedChar(line) {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// isRepeatedChar 检查是否为重复字符组成的行
func (is *IMAPService) isRepeatedChar(line string) bool {
	if len(line) < 10 {
		return false
	}

	// 检查是否主要由同一字符组成
	charCount := make(map[rune]int)
	for _, char := range line {
		charCount[char]++
	}

	// 如果某个字符占比超过80%，认为是重复字符行
	for _, count := range charCount {
		if float64(count)/float64(len(line)) > 0.8 {
			return true
		}
	}

	return false
}

// MarkAsRead 标记邮件为已读
func (is *IMAPService) MarkAsRead(emailID string) error {
	// 连接IMAP服务器
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", is.host, is.port), nil)
	if err != nil {
		return fmt.Errorf("连接IMAP服务器失败: %v", err)
	}
	defer c.Logout()

	// 登录
	if err := c.Login(is.username, is.password); err != nil {
		return fmt.Errorf("IMAP登录失败: %v", err)
	}

	// 选择收件箱
	if _, err := c.Select("INBOX", false); err != nil {
		return fmt.Errorf("选择收件箱失败: %v", err)
	}

	// 解析邮件序号
	seqNum, err := strconv.Atoi(emailID)
	if err != nil {
		return fmt.Errorf("无效的邮件ID: %v", err)
	}

	// 标记为已读
	seqset := new(imap.SeqSet)
	seqset.AddNum(uint32(seqNum))

	item := imap.FormatFlagsOp(imap.AddFlags, true)
	flags := []interface{}{imap.SeenFlag}
	if err := c.Store(seqset, item, flags, nil); err != nil {
		return fmt.Errorf("标记邮件已读失败: %v", err)
	}

	return nil
}

// TestConnection 测试IMAP连接
func (is *IMAPService) TestConnection() error {
	// 连接测试
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", is.host, is.port), nil)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer c.Logout()

	// 登录测试
	if err := c.Login(is.username, is.password); err != nil {
		return fmt.Errorf("认证失败: %v", err)
	}

	// 选择收件箱测试
	if _, err := c.Select("INBOX", true); err != nil {
		return fmt.Errorf("访问收件箱失败: %v", err)
	}

	return nil
}

// GetServiceInfo 获取IMAP服务信息
func (is *IMAPService) GetServiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"service_type": "IMAP",
		"host":         is.host,
		"port":         is.port,
		"username":     utils.MaskEmail(is.username),
		"status":       "active",
		"auth_method":  "app_password",
		"secure":       true,
	}
}

// extractTextFromHTML 从HTML中提取纯文本（更彻底的方法）
func (is *IMAPService) extractTextFromHTML(content string) string {
	// 移除整个HTML文档结构
	content = is.removeHTMLDocument(content)

	// 替换常见HTML标签为换行或空格
	replacements := map[string]string{
		"<br>":   "\n",
		"<br/>":  "\n",
		"<br />": "\n",
		"</p>":   "\n",
		"</div>": "\n",
		"</h1>":  "\n",
		"</h2>":  "\n",
		"</h3>":  "\n",
		"</li>":  "\n",
		"</tr>":  "\n",
		"</td>":  " ",
		"&nbsp;": " ",
	}

	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	// 移除所有剩余的HTML标签
	var result strings.Builder
	inTag := false

	for _, char := range content {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(char)
		}
	}

	return result.String()
}

// removeHTMLDocument 移除HTML文档结构，只保留body内容
func (is *IMAPService) removeHTMLDocument(content string) string {
	// 寻找body标签内容
	bodyStart := strings.Index(strings.ToLower(content), "<body")
	bodyEnd := strings.Index(strings.ToLower(content), "</body>")

	if bodyStart != -1 && bodyEnd != -1 {
		// 找到body标签的结束位置
		bodyContentStart := strings.Index(content[bodyStart:], ">")
		if bodyContentStart != -1 {
			bodyContentStart += bodyStart + 1
			return content[bodyContentStart:bodyEnd]
		}
	}

	// 如果没有找到body标签，返回原内容
	return content
}
