package email

import (
	"bytes"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/go-enols/go-log"
)

// ImapClient 封装的 IMAP 客户端
// 提供了一系列简化的方法用于邮件收发和管理
type ImapClient struct {
	client *imapclient.Client
}

// Connect 连接到 IMAP 服务器
// Deprecated: 请使用 AutoLogin 方法替代，此方法将在未来版本中移除
// 参数:
//   - address: 服务器地址，格式为 "host:port"，例如 "imap.example.com:993"
//   - username: 用户名/邮箱地址
//   - password: 密码或授权码
//
// 返回:
//   - *Client: 连接成功的客户端实例
//   - error: 连接过程中的错误，如果为nil则表示连接成功
func ImapConnect(address, username, password string) (*ImapClient, error) {
	client, err := imapclient.DialTLS(address, nil)
	if err != nil {
		return nil, err
	}

	err = client.Login(username, password).Wait()
	if err != nil {
		client.Close()
		return nil, err
	}

	return &ImapClient{
		client: client,
	}, nil
}

// Close 关闭与IMAP服务器的连接
// 应在使用完客户端后调用此方法释放资源
// 返回:
//   - error: 关闭连接过程中的错误，如果为nil则表示关闭成功
func (c *ImapClient) Close() error {
	return c.client.Close()
}

// ListMailboxes 获取所有邮箱列表
// 返回:
//   - []string: 邮箱名称列表，如"INBOX"、"Sent"、"Drafts"等
//   - error: 获取过程中的错误，如果为nil则表示获取成功
func (c *ImapClient) ListMailboxes() ([]string, error) {
	listCmd := c.client.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return nil, err
	}

	var names []string
	for _, m := range mailboxes {
		names = append(names, m.Mailbox)
	}

	return names, nil
}

// ParsedMessage 解析后的邮件结构
// 包含了邮件的各种元数据和内容
type ParsedMessage struct {
	MessageID    string          // 邮件ID
	Subject      string          // 邮件主题
	From         []*mail.Address // 发件人列表
	To           []*mail.Address // 收件人列表
	Cc           []*mail.Address // 抄送人列表
	InternalDate time.Time       // 邮件接收时间
	TextBody     string          // 纯文本格式的邮件正文
	HTMLBody     string          // HTML格式的邮件正文
	Flags        []imap.Flag     // 邮件标志，如已读、已回复等
	Attachments  []*Attachment   // 邮件附件列表
}

// Attachment 邮件附件结构
type Attachment struct {
	Filename    string // 附件文件名
	ContentType string // 附件内容类型，如"application/pdf"
	Data        []byte // 附件二进制数据
}

// parseMessage 解析邮件数据
// 内部方法，将IMAP邮件数据转换为更易使用的ParsedMessage结构
// 参数:
//   - msg: IMAP服务器返回的原始邮件数据
//
// 返回:
//   - *ParsedMessage: 解析后的邮件结构
//   - error: 解析过程中的错误
func parseMessage(msg *imapclient.FetchMessageData) (*ParsedMessage, error) {
	buf, err := msg.Collect()
	if err != nil {
		return nil, err
	}

	parsedMsg := &ParsedMessage{
		Flags:        buf.Flags,
		InternalDate: buf.InternalDate,
	}

	// 获取邮件信封信息
	if buf.Envelope != nil {
		parsedMsg.Subject = buf.Envelope.Subject
		parsedMsg.MessageID = buf.Envelope.MessageID

		// 转换发件人信息
		if len(buf.Envelope.From) > 0 {
			for _, addr := range buf.Envelope.From {
				mailAddr := &mail.Address{
					Name:    addr.Name,
					Address: addr.Mailbox + "@" + addr.Host,
				}
				parsedMsg.From = append(parsedMsg.From, mailAddr)
			}
		}

		// 转换收件人信息
		if len(buf.Envelope.To) > 0 {
			for _, addr := range buf.Envelope.To {
				mailAddr := &mail.Address{
					Name:    addr.Name,
					Address: addr.Mailbox + "@" + addr.Host,
				}
				parsedMsg.To = append(parsedMsg.To, mailAddr)
			}
		}

		// 转换抄送信息
		if len(buf.Envelope.Cc) > 0 {
			for _, addr := range buf.Envelope.Cc {
				mailAddr := &mail.Address{
					Name:    addr.Name,
					Address: addr.Mailbox + "@" + addr.Host,
				}
				parsedMsg.Cc = append(parsedMsg.Cc, mailAddr)
			}
		}
	}

	// 获取完整的邮件内容
	fullSection := &imap.FetchItemBodySection{}

	fullBody := buf.FindBodySection(fullSection)
	if fullBody == nil {
		// 尝试获取文本部分
		textSection := &imap.FetchItemBodySection{
			Specifier: imap.PartSpecifierText,
		}
		fullBody = buf.FindBodySection(textSection)
	}

	if fullBody == nil {
		return parsedMsg, nil
	}

	// 使用go-message解析邮件内容
	entity, err := message.Read(bytes.NewReader(fullBody))
	if err != nil {
		return parsedMsg, err
	}

	mr, err := mail.CreateReader(bytes.NewReader(fullBody))
	if err != nil {
		return parsedMsg, err
	}

	// 遍历解析各个部分
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// 处理正文内容
			contentType := h.Get("Content-Type")
			data, err := io.ReadAll(p.Body)
			if err != nil {
				continue
			}

			if strings.HasPrefix(contentType, "text/plain") {
				parsedMsg.TextBody = string(data)
			} else if strings.HasPrefix(contentType, "text/html") {
				parsedMsg.HTMLBody = string(data)
			}

		case *mail.AttachmentHeader:
			// 处理附件
			filename, _ := h.Filename()
			if filename == "" {
				filename = "attachment"
			}

			contentType := h.Get("Content-Type")
			data, err := io.ReadAll(p.Body)
			if err != nil {
				continue
			}

			parsedMsg.Attachments = append(parsedMsg.Attachments, &Attachment{
				Filename:    filename,
				ContentType: contentType,
				Data:        data,
			})
		}
	}

	// 如果正文为空，尝试从邮件实体中提取
	if parsedMsg.TextBody == "" && parsedMsg.HTMLBody == "" {
		textBody, htmlBody, _ := extractBodyFromEntity(entity)
		parsedMsg.TextBody = textBody
		parsedMsg.HTMLBody = htmlBody
	}
	log.Debug("主题", parsedMsg.Subject)
	return parsedMsg, nil
}

// extractBodyFromEntity 从邮件实体中提取正文
// 内部方法，用于处理复杂的多部分邮件，提取文本和HTML正文
// 参数:
//   - e: 邮件实体
//
// 返回:
//   - textBody: 提取的纯文本正文
//   - htmlBody: 提取的HTML正文
//   - err: 提取过程中的错误
func extractBodyFromEntity(e *message.Entity) (textBody string, htmlBody string, err error) {
	// 如果是多部分邮件
	if mr := e.MultipartReader(); mr != nil {
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}

			mediaType, _, _ := p.Header.ContentType()

			if mediaType == "text/plain" {
				data, _ := io.ReadAll(p.Body)
				textBody = string(data)
			} else if mediaType == "text/html" {
				data, _ := io.ReadAll(p.Body)
				htmlBody = string(data)
			} else if strings.HasPrefix(mediaType, "multipart/") {
				// 递归处理嵌套的多部分邮件
				nestedText, nestedHTML, _ := extractBodyFromEntity(p)
				if textBody == "" {
					textBody = nestedText
				}
				if htmlBody == "" {
					htmlBody = nestedHTML
				}
			}
		}
	} else {
		// 处理单一部分邮件
		mediaType, _, _ := e.Header.ContentType()

		if mediaType == "text/plain" {
			data, _ := io.ReadAll(e.Body)
			textBody = string(data)
		} else if mediaType == "text/html" {
			data, _ := io.ReadAll(e.Body)
			htmlBody = string(data)
		}
	}

	return
}

// GetEmail 获取指定邮箱中的最近几封邮件
// 可以通过可选参数指定邮件数量和邮箱名称
// 参数(通过opt ...any传递):
//   - int: [可选] 要获取的邮件数量，默认为10封
//   - string: [可选] 邮箱名称，默认为"INBOX"(收件箱)
//
// 用法示例:
//   - client.GetEmail() - 获取收件箱最新的10封邮件
//   - client.GetEmail(5) - 获取收件箱最新的5封邮件
//   - client.GetEmail("Sent") - 获取已发送邮件箱中最新的10封邮件
//   - client.GetEmail(3, "Drafts") - 获取草稿箱中最新的3封邮件
//
// 返回:
//   - []*ParsedMessage: 解析后的邮件列表
//   - error: 获取过程中的错误
func (c *ImapClient) GetEmail(opt ...any) ([]*ParsedMessage, error) {
	var n = 10
	var mailbox = "INBOX"
	// 处理传入的参数
	for _, v := range opt {
		switch val := v.(type) {
		case int:
			n = val
		case string:
			mailbox = val
		}
	}
	// 选择邮箱
	_, err := c.client.Select(mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, err
	}

	status, err := c.client.Status(mailbox, &imap.StatusOptions{
		NumMessages: true,
	}).Wait()
	if err != nil {
		return nil, err
	}

	var numMessages int
	if status.NumMessages != nil {
		numMessages = int(*status.NumMessages)
	} else {
		numMessages = 0
	}

	start := numMessages - n + 1
	if start < 1 {
		start = 1
	}

	seqSet := imap.SeqSet{
		{
			Start: uint32(start),
			Stop:  uint32(numMessages),
		},
	}

	// 获取邮件完整内容
	cmd := c.client.Fetch(seqSet, &imap.FetchOptions{
		Flags:        true,
		InternalDate: true,
		RFC822Size:   true,
		Envelope:     true,
		BodySection: []*imap.FetchItemBodySection{
			{}, // 获取完整邮件
		},
	})

	var messages []*ParsedMessage
	for msg := cmd.Next(); msg != nil; msg = cmd.Next() {
		parsedMsg, err := parseMessage(msg)
		if err != nil {
			return nil, err
		}
		messages = append(messages, parsedMsg)
	}

	return messages, nil
}

// MonitEmail 监听新邮件到来，当收到指定数量的新邮件或超时后返回
// 参数(通过opt ...any传递):
//   - int: [可选] 要等待的新邮件数量，默认为1封
//   - string: [可选] 要监听的邮箱名称，默认为"INBOX"(收件箱)
//   - time.Duration: [可选] 监听超时时间，默认为5分钟
//
// 用法示例:
//   - client.MonitEmail() - 监听收件箱，等待1封新邮件或5分钟超时
//   - client.MonitEmail(3) - 监听收件箱，等待3封新邮件或5分钟超时
//   - client.MonitEmail("Sent") - 监听已发送邮件箱，等待1封新邮件或5分钟超时
//   - client.MonitEmail(2, "INBOX", time.Minute*10) - 监听收件箱，等待2封新邮件或10分钟超时
//
// 返回:
//   - []*ParsedMessage: 监听期间收到的新邮件列表
//   - error: 监听过程中的错误
func (c *ImapClient) MonitEmail(opt ...any) ([]*ParsedMessage, error) {
	var n = 1
	var mailbox = "INBOX"
	var timeout = time.Minute * 5 // 默认监听超时时间为5分钟

	// 处理传入的参数
	for _, v := range opt {
		switch val := v.(type) {
		case int:
			n = val
		case string:
			mailbox = val
		case time.Duration:
			timeout = val
		}
	}

	// 选择邮箱
	_, err := c.client.Select(mailbox, &imap.SelectOptions{ReadOnly: true}).Wait()
	if err != nil {
		return nil, err
	}

	// 获取初始邮件数量
	status, err := c.client.Status(mailbox, &imap.StatusOptions{
		NumMessages: true,
	}).Wait()
	if err != nil {
		return nil, err
	}

	var initialMessages uint32 = 0
	if status.NumMessages != nil {
		initialMessages = *status.NumMessages
	}

	log.Debug("当前邮箱有", initialMessages, "封邮件，开始监听新邮件")

	// 启动IDLE命令进行监听
	idleCmd, err := c.client.Idle()
	if err != nil {
		return nil, err
	}

	// 设置超时
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// 用于接收终止信号
	stopChan := make(chan struct{})

	// 用于存储新邮件
	var newMessages []*ParsedMessage

	// 在goroutine中检查邮箱更新
	go func() {
		for {
			select {
			case <-timer.C:
				// 超时，关闭IDLE命令
				idleCmd.Close()
				return

			case <-stopChan:
				// 收到停止信号
				return

			case <-time.After(time.Second * 3):
				// 每3秒检查一次邮箱状态
				currentStatus, err := c.client.Status(mailbox, &imap.StatusOptions{
					NumMessages: true,
				}).Wait()
				if err != nil {
					continue
				}

				if currentStatus.NumMessages == nil {
					continue
				}

				currentMessages := *currentStatus.NumMessages
				if currentMessages > initialMessages {
					// 有新邮件到达
					newCount := currentMessages - initialMessages
					log.Debug("检测到", newCount, "封新邮件")

					// 暂时中断IDLE
					idleCmd.Close()

					// 获取新邮件
					seqSet := imap.SeqSet{}
					for i := initialMessages + 1; i <= currentMessages; i++ {
						seqSet.AddNum(i)
					}

					// 获取新邮件的完整内容
					fetchCmd := c.client.Fetch(seqSet, &imap.FetchOptions{
						Flags:        true,
						InternalDate: true,
						RFC822Size:   true,
						Envelope:     true,
						BodySection: []*imap.FetchItemBodySection{
							{}, // 获取完整邮件
						},
					})

					// 解析新邮件
					for msg := fetchCmd.Next(); msg != nil; msg = fetchCmd.Next() {
						parsedMsg, err := parseMessage(msg)
						if err != nil {
							continue
						}
						newMessages = append(newMessages, parsedMsg)
					}

					if err := fetchCmd.Close(); err != nil {
						log.Debug("获取邮件时出错:", err)
					}

					// 更新初始计数
					initialMessages = currentMessages

					// 如果达到所需数量，就停止监听
					if len(newMessages) >= n {
						close(stopChan)
						return
					}

					// 重新开始IDLE
					idleCmd, err = c.client.Idle()
					if err != nil {
						close(stopChan)
						return
					}
				}
			}
		}
	}()

	// 等待监听结束
	select {
	case <-stopChan:
		// 收到停止信号
		if idleCmd != nil {
			idleCmd.Close()
		}

	case <-timer.C:
		// 超时
		if idleCmd != nil {
			idleCmd.Close()
		}
		log.Debug("监听超时，收到", len(newMessages), "封新邮件")
	}

	// 等待IDLE命令完成
	if idleCmd != nil {
		idleCmd.Wait()
	}

	return newMessages, nil
}
