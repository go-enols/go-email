# go-email

一个功能强大的Go语言邮件客户端库，支持IMAP、POP3和SMTP协议。

## 特性

- 支持IMAP、POP3和SMTP协议
- 简洁易用的API接口
- 支持邮件发送、接收和管理
- 支持附件处理
- 符合Go语言设计哲学的架构

## 安装

```bash
go get github.com/go-enols/go-email
```

## 快速开始

### 发送邮件（SMTP）

```go
package main

import (
	"github.com/go-enols/go-email"
	"github.com/go-enols/go-log"
)

func main() {
	// 创建SMTP客户端
	client, err := email.AutoLoginSender(email.LoginParams{
		Host:  "smtp.qq.com",
		Port:  587,
		User:  "your-email@qq.com",
		Pwd:   "your-password",
		Proto: email.SMTP,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 发送邮件
	err = client.SendEmail(
		[]string{"recipient@example.com"}, // 收件人
		"测试邮件",                      // 主题
		"这是一封测试邮件",                // 正文
		[]*email.Attachment{},         // 附件
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("邮件发送成功")
}
```

### 读取邮件（IMAP）

```go
package main

import (
	"github.com/go-enols/go-email"
	"github.com/go-enols/go-log"
)

func main() {
	// 创建IMAP客户端
	client, err := email.AutoLoginReader(email.LoginParams{
		Host:  "imap.qq.com",
		Port:  993,
		User:  "your-email@qq.com",
		Pwd:   "your-password",
		Proto: email.IMAP,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 获取邮件
	messages, err := client.GetEmail(10) // 获取最新10封邮件
	if err != nil {
		log.Fatal(err)
	}

	log.Info("获取到", len(messages), "封邮件")
}
```

### 读取邮件（POP3）

```go
package main

import (
	"github.com/go-enols/go-email"
	"github.com/go-enols/go-log"
)

func main() {
	// 创建POP3客户端
	client, err := email.AutoLoginReader(email.LoginParams{
		Host:  "pop.qq.com",
		Port:  995,
		User:  "your-email@qq.com",
		Pwd:   "your-password",
		Proto: email.POP3,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// 获取邮件
	messages, err := client.GetEmail(10) // 获取最新10封邮件
	if err != nil {
		log.Fatal(err)
	}

	log.Info("获取到", len(messages), "封邮件")
}
```

## 架构设计

本库采用了符合Go语言设计哲学的架构：

1. **小接口**：使用小接口促进解耦，便于测试和复用
2. **避免包级状态**：使用结构体封装状态，提高并发安全性
3. **清晰的代码结构**：避免使用复杂的技巧，保持代码可读性

## 接口设计

### EmailClient

基础邮件客户端接口，定义了关闭连接的方法：

```go
type EmailClient interface {
	Close() error
}
```

### EmailReader

邮件读取接口，继承自EmailClient，定义了读取邮件的方法：

```go
type EmailReader interface {
	EmailClient
	GetEmail(opt ...any) ([]*ParsedMessage, error)
	ListMailboxes() ([]string, error)
	MonitEmail(opt ...any) ([]*ParsedMessage, error)
}
```

### EmailSender

邮件发送接口，继承自EmailClient，定义了发送邮件的方法：

```go
type EmailSender interface {
	EmailClient
	SendEmail(to []string, subject, body string, attachments []*Attachment) error
}
```

## 最佳实践

1. **使用AutoLoginReader和AutoLoginSender**：根据需要选择合适的客户端类型
2. **总是使用defer关闭连接**：确保资源正确释放
3. **处理错误**：及时处理可能的错误
4. **使用结构体封装状态**：避免使用全局变量
5. **保持代码简洁**：遵循Go语言的设计哲学

## 示例

完整的示例代码位于`example`目录中，包含了IMAP、POP3和SMTP的使用示例。

## 许可证

MIT

