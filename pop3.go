package email

import (
	"fmt"
	"io"
	"mime"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/go-enols/go-log"
	"github.com/knadh/go-pop3"
)

// POP3Client 使用github.com/knadh/go-pop3库的POP3客户端结构体
type POP3Client struct {
	client *pop3.Client
	conn   *pop3.Conn
	host   string
	port   int
	user   string
	pwd    string
}

// NewPOP3Client 创建新的POP3客户端
func NewPOP3Client(host string, port int, user, pwd string) *POP3Client {
	return &POP3Client{
		host: host,
		port: port,
		user: user,
		pwd:  pwd,
	}
}

// Connect 连接并登录到POP3服务器
func (c *POP3Client) Connect() error {
	// 创建POP3客户端
	c.client = pop3.New(pop3.Opt{
		Host:       c.host,
		Port:       c.port,
		TLSEnabled: true,
	})

	// 创建连接
	conn, err := c.client.NewConn()
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	c.conn = conn

	// 登录
	err = c.conn.Auth(c.user, c.pwd)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %v", err)
	}

	return nil
}

// Close 关闭连接
func (c *POP3Client) Close() error {
	if c.conn != nil {
		return c.conn.Quit()
	}
	return nil
}

// GetEmail 获取邮件列表
// 参数:
//   - opt: 可选参数，支持int类型指定获取邮件数量（默认10封）
//
// 返回:
//   - []*ParsedMessage: 解析后的邮件列表
//   - error: 获取过程中的错误
func (c *POP3Client) GetEmail(opt ...any) ([]*ParsedMessage, error) {
	var n = 10
	// 处理传入的参数
	for _, v := range opt {
		switch val := v.(type) {
		case int:
			n = val
		}
	}

	// 获取邮件数量和大小信息
	count, _, err := c.conn.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %v", err)
	}

	if count == 0 {
		return []*ParsedMessage{}, nil
	}

	// 计算要获取的邮件范围（获取最新的n封邮件）
	start := count - n + 1
	if start < 1 {
		start = 1
	}

	var messages []*ParsedMessage

	// 获取指定范围的邮件
	for i := start; i <= count; i++ {
		// 获取邮件内容
		msgEntity, err := c.conn.Retr(i)
		if err != nil {
			log.Error("Failed to retrieve message", i, ":", err)
			continue
		}

		// 解析邮件
		parsedMsg, err := c.parseMessage(msgEntity, i)
		if err != nil {
			log.Error("Failed to parse message", i, ":", err)
			continue
		}
		log.Debug("主题", parsedMsg.Subject)

		messages = append(messages, parsedMsg)
	}

	return messages, nil
}

// parseMessage 解析邮件内容
func (c *POP3Client) parseMessage(msg *message.Entity, msgNum int) (*ParsedMessage, error) {

	header := msg.Header

	// 解码Subject字段（处理RFC 2047编码）
	subject := header.Get("Subject")
	if subject != "" {
		decoder := mime.WordDecoder{}
		decodedSubject, err := decoder.DecodeHeader(subject)
		if err != nil {
			log.Error("Failed to decode subject:", err, "Original subject:", subject)
			// 如果解码失败，保持原始subject
		} else {
			// 只有在解码成功且结果不同时才更新
			if decodedSubject != subject {
				subject = decodedSubject
			}
		}
	}

	parsedMsg := &ParsedMessage{
		MessageID:    header.Get("Message-ID"),
		Subject:      subject,
		InternalDate: time.Now(),    // POP3不提供接收时间，使用当前时间
		Flags:        []imap.Flag{}, // POP3不支持标志
	}

	// 解析发件人
	fromHeader := header.Get("From")
	if fromHeader != "" {
		addresses, err := mail.ParseAddressList(fromHeader)
		if err == nil && len(addresses) > 0 {
			parsedMsg.From = []*mail.Address{addresses[0]}
		}
	}

	// 解析收件人
	toHeader := header.Get("To")
	if toHeader != "" {
		addresses, err := mail.ParseAddressList(toHeader)
		if err == nil {
			parsedMsg.To = addresses
		}
	}

	// 解析抄送
	ccHeader := header.Get("Cc")
	if ccHeader != "" {
		addresses, err := mail.ParseAddressList(ccHeader)
		if err == nil {
			parsedMsg.Cc = addresses
		}
	}

	// 解析邮件正文和附件
	body, attachments, err := c.parseMessageBody(msg)
	if err != nil {
		log.Error("Failed to parse message body:", err)
	}

	// 根据邮件内容类型设置正文
	if strings.Contains(strings.ToLower(body), "<html") || strings.Contains(strings.ToLower(body), "<body") {
		parsedMsg.HTMLBody = body
	} else {
		parsedMsg.TextBody = body
	}
	parsedMsg.Attachments = attachments

	return parsedMsg, nil
}

// parseMessageBody 解析邮件正文和附件
func (c *POP3Client) parseMessageBody(msg *message.Entity) (string, []*Attachment, error) {
	var body string
	var attachments []*Attachment

	// 检查Content-Type
	contentType, _, err := msg.Header.ContentType()
	if err != nil {
		contentType = "text/plain"
	}

	if strings.HasPrefix(contentType, "multipart/") {
		// 处理多部分邮件
		mr := msg.MultipartReader()
		if mr != nil {
			for {
				part, err := mr.NextPart()
				if err != nil {
					break
				}

				partContentType, partParams, _ := part.Header.ContentType()
				disposition, dispParams, _ := part.Header.ContentDisposition()

				if disposition == "attachment" || (disposition == "" && partParams["name"] != "") {
					// 这是一个附件
					attachment, err := c.parseAttachment(part, partContentType, dispParams, partParams)
					if err == nil {
						attachments = append(attachments, attachment)
					}
				} else if strings.HasPrefix(partContentType, "text/") {
					// 这是邮件正文
					partBody, err := c.readPartContent(part)
					if err == nil {
						if body == "" {
							body = partBody
						} else {
							body += "\n" + partBody
						}
					}
				}
			}
		}
	} else {
		// 处理单部分邮件
		body, err = c.readPartContent(msg)
		if err != nil {
			return "", nil, err
		}
	}

	return body, attachments, nil
}

// readPartContent 读取邮件部分的内容
func (c *POP3Client) readPartContent(part *message.Entity) (string, error) {
	body := part.Body

	// 读取所有内容
	content, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// parseAttachment 解析附件
func (c *POP3Client) parseAttachment(part *message.Entity, contentType string, dispParams, partParams map[string]string) (*Attachment, error) {
	// 获取文件名
	filename := dispParams["filename"]
	if filename == "" {
		filename = partParams["name"]
	}
	if filename == "" {
		filename = "attachment"
	}

	// 读取附件数据
	data, err := io.ReadAll(part.Body)
	if err != nil {
		return nil, err
	}

	return &Attachment{
		Filename:    filename,
		ContentType: contentType,
		Data:        data,
	}, nil
}

// ListMailboxes 列出邮箱（POP3不支持邮箱概念，返回空列表）
func (c *POP3Client) ListMailboxes() ([]string, error) {
	// POP3协议不支持邮箱概念，只有收件箱
	return []string{"INBOX"}, nil
}

// MonitEmail 监控新邮件（POP3不支持推送，使用轮询方式）
func (c *POP3Client) MonitEmail(opt ...any) ([]*ParsedMessage, error) {
	// POP3不支持实时监控，直接调用GetEmail获取邮件
	return c.GetEmail(opt...)
}
