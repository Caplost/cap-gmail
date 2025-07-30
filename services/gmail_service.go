package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gmail-forwarding-system/config"
	"gmail-forwarding-system/utils"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// EmailData 表示邮件数据
type EmailData struct {
	ID           string    `json:"id"`            // 邮件序号或ID
	MessageID    string    `json:"message_id"`    // 邮件唯一标识符
	From         string    `json:"from"`          // 发件人
	Subject      string    `json:"subject"`       // 主题
	Body         string    `json:"body"`          // 正文
	ReceivedTime time.Time `json:"received_time"` // 接收时间
}

// GmailService Gmail服务结构
type GmailService struct {
	service *gmail.Service
}

// NewGmailService 创建Gmail服务实例
func NewGmailService() (*GmailService, error) {
	ctx := context.Background()
	b, err := os.ReadFile(config.AppConfig.GmailCredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("无法读取客户端密钥文件: %v", err)
	}

	config, err := google.ConfigFromJSON(b, gmail.GmailReadonlyScope, gmail.GmailModifyScope, gmail.GmailSendScope)
	if err != nil {
		return nil, fmt.Errorf("无法解析客户端密钥文件: %v", err)
	}

	client := getClient(config)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("无法创建Gmail客户端: %v", err)
	}

	return &GmailService{service: srv}, nil
}

// GetUnreadEmails 获取未读邮件
func (gs *GmailService) GetUnreadEmails() ([]*EmailData, error) {
	user := "me"

	r, err := gs.service.Users.Messages.List(user).Q("is:unread").Do()
	if err != nil {
		return nil, fmt.Errorf("无法获取邮件列表: %v", err)
	}

	var emails []*EmailData
	for _, m := range r.Messages {
		email, err := gs.GetEmailByID(m.Id)
		if err != nil {
			utils.LogWarn(fmt.Sprintf("获取邮件详情失败: %v", err))
			continue
		}
		emails = append(emails, email)
	}

	return emails, nil
}

// GetEmailByID 根据ID获取邮件详情
func (gs *GmailService) GetEmailByID(messageID string) (*EmailData, error) {
	user := "me"

	m, err := gs.service.Users.Messages.Get(user, messageID).Do()
	if err != nil {
		return nil, fmt.Errorf("获取邮件失败: %v", err)
	}

	var from, subject, body string

	for _, header := range m.Payload.Headers {
		switch header.Name {
		case "From":
			from = header.Value
		case "Subject":
			subject = header.Value
		}
	}

	body = getEmailBody(m.Payload)

	email := &EmailData{
		ID:           messageID,
		MessageID:    messageID,
		From:         from,
		Subject:      subject,
		Body:         body,
		ReceivedTime: time.Now(),
	}

	return email, nil
}

// ForwardEmail 转发邮件
func (gs *GmailService) ForwardEmail(email *EmailData, targetEmail string) error {
	// 🔧 Gmail API发送前检查并强制解码邮件内容
	fmt.Printf("\n📤 Gmail API发送前解码检查:\n")
	fmt.Printf("  原始Body: %s\n", truncateGmailString(email.Body, 100))

	// 检查是否包含编码字符
	hasEncoding := strings.Contains(email.Body, "=E") ||
		strings.Contains(email.Body, "=C") ||
		strings.Contains(email.Body, "=D")
	fmt.Printf("  检测到编码字符: %v\n", hasEncoding)

	// 如果检测到编码，强制解码
	if hasEncoding {
		decodedBody := utils.DecodeEmailContent(email.Body, "quoted-printable")
		fmt.Printf("  强制解码后Body: %s\n", truncateGmailString(decodedBody, 100))
		fmt.Printf("  解码是否成功: %v\n", decodedBody != email.Body)
		email.Body = decodedBody
	} else {
		fmt.Printf("  无编码字符，跳过解码\n")
	}
	fmt.Printf("=====================================\n\n")

	// 使用标准的转发主题格式
	forwardSubject := fmt.Sprintf("Fwd: %s", email.Subject)

	// 使用标准转发格式（与SMTP保持一致）
	forwardBody := fmt.Sprintf(`

---------- Forwarded message ---------
From: %s
Subject: %s
Date: %s

%s`, email.From, email.Subject, email.ReceivedTime.Format("2006-01-02 15:04:05"), email.Body)

	return gs.SendEmail(targetEmail, forwardSubject, forwardBody)
}

// SendEmail 发送邮件
func (gs *GmailService) SendEmail(to, subject, body string) error {
	user := "me"

	message := createMessage(to, subject, body)
	_, err := gs.service.Users.Messages.Send(user, message).Do()
	if err != nil {
		return fmt.Errorf("发送邮件失败: %v", err)
	}

	return nil
}

// MarkAsRead 标记邮件为已读
func (gs *GmailService) MarkAsRead(emailID string) error {
	// Gmail API使用消息ID
	messageID := emailID
	user := "me"

	modifyRequest := &gmail.ModifyMessageRequest{
		AddLabelIds:    []string{},
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err := gs.service.Users.Messages.Modify(user, messageID, modifyRequest).Do()
	if err != nil {
		return fmt.Errorf("标记邮件已读失败: %v", err)
	}

	return nil
}

// GetServiceInfo 获取Gmail服务信息
func (gs *GmailService) GetServiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"service_type": "Gmail_API",
		"auth_method":  "OAuth2",
		"status":       "active",
		"secure":       true,
	}
}

// 辅助函数保持不变
func getClient(oauthConfig *oauth2.Config) *http.Client {
	tokFile := config.AppConfig.GmailTokenPath
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(oauthConfig)
		saveToken(tokFile, tok)
	}
	return oauthConfig.Client(context.Background(), tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("访问以下URL进行授权，然后输入授权码: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("无法读取授权码: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("无法获取令牌: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("保存令牌到: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("无法保存令牌: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

func getEmailBody(payload *gmail.MessagePart) string {
	if payload.Body != nil && payload.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(payload.Body.Data)
		if err == nil {
			rawContent := string(data)

			// 检测Content-Transfer-Encoding
			encoding := ""
			if payload.Headers != nil {
				for _, header := range payload.Headers {
					if strings.ToLower(header.Name) == "content-transfer-encoding" {
						encoding = strings.ToLower(header.Value)
						break
					}
				}
			}

			// 使用通用解码器处理邮件内容
			return utils.DecodeEmailContent(rawContent, encoding)
		}
	}

	// 递归查找正文
	for _, part := range payload.Parts {
		if body := getEmailBody(part); body != "" {
			return body
		}
	}

	return ""
}

func createMessage(to, subject, body string) *gmail.Message {
	message := fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", to, subject, body)
	return &gmail.Message{
		Raw: base64.URLEncoding.EncodeToString([]byte(message)),
	}
}

// truncateGmailString 截断字符串用于Gmail日志显示
func truncateGmailString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
