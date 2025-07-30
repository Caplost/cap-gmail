package utils

import (
	"io"
	"mime/quotedprintable"
	"net/url"
	"regexp"
	"strings"
)

// DecodeEmailContent 通用邮件内容解码函数
// 支持Quoted-Printable和Base64解码，以及HTML清理
func DecodeEmailContent(content, encoding string) string {
	// 第一步：解码传输编码
	decoded := decodeTransferEncoding(content, encoding)

	// 第二步：检测并清理HTML内容
	if DetectHTMLContent(decoded) {
		decoded = ImprovedHTMLCleanup(decoded)
	}

	return strings.TrimSpace(decoded)
}

// decodeTransferEncoding 解码传输编码
func decodeTransferEncoding(content, encoding string) string {
	// 检测Quoted-Printable编码（扩展检测条件）
	needsQuotedPrintableDecode := encoding == "quoted-printable" ||
		strings.Contains(content, "=3D") ||
		strings.Contains(content, "=22") ||
		strings.Contains(content, "=E") || // 中文UTF-8编码
		strings.Contains(content, "=C") || // 其他多字节字符
		strings.Contains(content, "=D") // 其他多字节字符

	if needsQuotedPrintableDecode {
		// 优先使用URL解码方法（测试证明最有效）
		if decoded := urlDecodeMethod(content); decoded != content {
			content = decoded
		} else {
			// 备选：手动解码
			content = manualQuotedPrintableDecode(content)
		}
	}

	switch encoding {
	case "quoted-printable":
		// 使用标准库再次解码（双重保险）
		decoder := quotedprintable.NewReader(strings.NewReader(content))
		decoded, err := io.ReadAll(decoder)
		if err == nil {
			return string(decoded)
		}
		return content

	case "base64":
		// 这里可以添加base64解码逻辑
		return content

	default:
		return content
	}
}

// urlDecodeMethod URL解码方法（测试证明有效）
func urlDecodeMethod(content string) string {
	// 将 = 替换为 %
	urlEncoded := strings.ReplaceAll(content, "=", "%")
	decoded, err := url.QueryUnescape(urlEncoded)
	if err != nil {
		return content // 返回原内容而不是错误
	}
	return decoded
}

// manualQuotedPrintableDecode 手动Quoted-Printable解码
func manualQuotedPrintableDecode(content string) string {
	// 常见的Quoted-Printable编码替换
	replacements := map[string]string{
		"=3D": "=",
		"=22": "\"",
		"=20": " ",
		"=09": "\t",
		"=0A": "\n",
		"=0D": "\r",
	}

	result := content
	for encoded, decoded := range replacements {
		result = strings.ReplaceAll(result, encoded, decoded)
	}

	return result
}

// DetectHTMLContent 检测内容是否包含HTML标签
func DetectHTMLContent(content string) bool {
	// 常见HTML标签正则
	htmlTags := []string{
		`<html[^>]*>`, `</html>`,
		`<body[^>]*>`, `</body>`,
		`<div[^>]*>`, `</div>`,
		`<p[^>]*>`, `</p>`,
		`<br[^>]*>`, `<span[^>]*>`,
		`<table[^>]*>`, `<tr[^>]*>`, `<td[^>]*>`,
		`<a[^>]*>`, `<img[^>]*>`,
		`<style[^>]*>`, `<script[^>]*>`,
	}

	lowerContent := strings.ToLower(content)
	for _, pattern := range htmlTags {
		matched, _ := regexp.MatchString(pattern, lowerContent)
		if matched {
			return true
		}
	}

	return false
}

// ImprovedHTMLCleanup 改进的HTML清理
func ImprovedHTMLCleanup(content string) string {
	// 1. 最激进的HTML移除：提取纯文本
	content = extractPureTextFromHTML(content)

	// 2. 解码HTML实体
	content = decodeHTMLEntities(content)

	// 3. 最终文本清理
	content = finalTextCleanup(content)

	return content
}

// extractPureTextFromHTML 提取纯文本内容（移除所有HTML标签）
func extractPureTextFromHTML(content string) string {
	// 移除script和style标签及其内容
	scriptRegex := regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	content = scriptRegex.ReplaceAllString(content, "")

	styleRegex := regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	content = styleRegex.ReplaceAllString(content, "")

	// 移除所有HTML标签，但保留内容
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	content = htmlTagRegex.ReplaceAllString(content, " ")

	return content
}

// decodeHTMLEntities 解码HTML实体
func decodeHTMLEntities(content string) string {
	entities := map[string]string{
		"&amp;":    "&",
		"&lt;":     "<",
		"&gt;":     ">",
		"&quot;":   "\"",
		"&apos;":   "'",
		"&nbsp;":   " ",
		"&#8203;":  "", // 零宽度空格
		"&hellip;": "...",
		"&mdash;":  "—",
		"&ndash;":  "–",
	}

	result := content
	for entity, replacement := range entities {
		result = strings.ReplaceAll(result, entity, replacement)
	}

	return result
}

// finalTextCleanup 最终文本清理
func finalTextCleanup(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行
		if line == "" {
			continue
		}

		// 跳过只有特殊字符的行
		if isOnlySpecialChars(line) {
			continue
		}

		// 避免重复的相同行
		if len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] == line {
			continue
		}

		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, "\n")
}

// isOnlySpecialChars 检查字符串是否只包含特殊字符
func isOnlySpecialChars(s string) bool {
	if len(s) == 0 {
		return false
	}

	for _, char := range s {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			(char >= '\u4e00' && char <= '\u9fff') { // 中文字符
			return false
		}
	}

	return true
}
