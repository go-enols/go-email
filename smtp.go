package email

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/smtp"
	"strings"
)

// SMTPClient SMTP客户端结构体
type SMTPClient struct {
	auth smtp.Auth
	host string
	port int
	user string
}

// 确保SMTPClient实现了EmailSender接口
var _ EmailSender = (*SMTPClient)(nil)

// Close 关闭SMTP连接（SMTP是无状态的，此方法为空实现）
func (c *SMTPClient) Close() error {
	return nil
}

// GetEmail SMTP不支持获取邮件，返回空列表
func (c *SMTPClient) GetEmail(opt ...any) ([]*ParsedMessage, error) {
	return []*ParsedMessage{}, nil
}

// ListMailboxes SMTP不支持邮箱列表，返回空列表
func (c *SMTPClient) ListMailboxes() ([]string, error) {
	return []string{}, nil
}

// MonitEmail SMTP不支持监控邮件，返回空列表
func (c *SMTPClient) MonitEmail(opt ...any) ([]*ParsedMessage, error) {
	return []*ParsedMessage{}, nil
}

// SendEmail 发送邮件
// 参数:
//   - to: 收件人邮箱列表
//   - subject: 邮件主题
//   - body: 邮件正文
//   - attachments: 附件列表
//
// 返回:
//   - error: 发送过程中的错误
func (c *SMTPClient) SendEmail(to []string, subject, body string, attachments []*Attachment) error {
	// 构建邮件头
	from := c.user
	msg := &bytes.Buffer{}
	w := multipart.NewWriter(msg)

	// 邮件头
	headers := map[string]string{
		"From":         from,
		"To":           strings.Join(to, ", "),
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s", w.Boundary()),
	}

	for k, v := range headers {
		fmt.Fprintf(msg, "%s: %s\r\n", k, v)
	}
	fmt.Fprintf(msg, "\r\n")

	// 邮件正文
	part, err := w.CreatePart(map[string][]string{
		"Content-Type": {"text/plain; charset=utf-8"},
	})
	if err != nil {
		return err
	}
	part.Write([]byte(body))

	// 附件
	for _, attachment := range attachments {
		part, err := w.CreatePart(map[string][]string{
			"Content-Type":        {attachment.ContentType},
			"Content-Disposition": {fmt.Sprintf("attachment; filename=%s", attachment.Filename)},
		})
		if err != nil {
			return err
		}
		part.Write(attachment.Data)
	}

	// 结束邮件
	w.Close()

	// 发送邮件
	auth := c.auth
	serverAddr := fmt.Sprintf("%s:%d", c.host, c.port)
	return smtp.SendMail(serverAddr, auth, from, to, msg.Bytes())
}

// NewSMTPClient 创建新的SMTP客户端
func NewSMTPClient(host string, port int, user, pwd string) *SMTPClient {
	auth := smtp.PlainAuth("", user, pwd, host)
	return &SMTPClient{
		auth: auth,
		host: host,
		port: port,
		user: user,
	}
}
