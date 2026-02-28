package email

import (
	"fmt"

	"github.com/emersion/go-imap/v2/imapclient"
)

// EmailClient 基础邮件客户端接口
type EmailClient interface {
	Close() error
}

// EmailReader 邮件读取接口
type EmailReader interface {
	EmailClient
	GetEmail(opt ...any) ([]*ParsedMessage, error)
	ListMailboxes() ([]string, error)
	MonitEmail(opt ...any) ([]*ParsedMessage, error)
}

// EmailSender 邮件发送接口
type EmailSender interface {
	EmailClient
	SendEmail(to []string, subject, body string, attachments []*Attachment) error
}

// createIMAPClient 创建IMAP客户端
func createIMAPClient(data LoginParams) (EmailReader, error) {
	client, err := imapclient.DialTLS(fmt.Sprintf("%s:%d", data.Host, data.Port), nil)
	if err != nil {
		return nil, err
	}

	switch data.Host {
	case "outlook.office365.com":
		token, err := getAccessTokenFromRefreshToken(data.RefreshToken, data.ClientId)
		if err != nil {
			return nil, err
		}
		if token["code"].(int) != 0 {
			return nil, err
		}

		accessToken := token["access_token"].(string)
		auth := &XOAUTH2Authenticator{
			Username:    data.User,
			AccessToken: accessToken,
		}
		if err := client.Authenticate(auth); err != nil {
			return nil, err
		}
	default:
		err = client.Login(data.User, data.Pwd).Wait()
		if err != nil {
			client.Close()
			return nil, err
		}
	}
	return &ImapClient{
		client: client,
	}, nil
}

// createPOP3Client 创建POP3客户端
func createPOP3Client(data LoginParams) (EmailReader, error) {
	// 创建POP3客户端实例
	client := NewPOP3Client(data.Host, data.Port, data.User, data.Pwd)

	// 连接并登录
	err := client.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to POP3 server: %v", err)
	}

	return client, nil
}

// createSMTPClient 创建SMTP客户端
func createSMTPClient(data LoginParams) (EmailSender, error) {
	client := NewSMTPClient(data.Host, data.Port, data.User, data.Pwd)
	return client, nil
}

// AutoLogin 根据协议类型自动创建对应的客户端
func AutoLogin(data LoginParams) (EmailClient, error) {
	switch data.Proto {
	case IMAP:
		return createIMAPClient(data)
	case POP3:
		return createPOP3Client(data)
	case SMTP:
		return createSMTPClient(data)
	default:
		return nil, fmt.Errorf("unsupported protocol: %d", data.Proto)
	}
}

// AutoLoginReader 创建邮件读取客户端
func AutoLoginReader(data LoginParams) (EmailReader, error) {
	switch data.Proto {
	case IMAP, POP3:
		client, err := AutoLogin(data)
		if err != nil {
			return nil, err
		}
		return client.(EmailReader), nil
	default:
		return nil, fmt.Errorf("protocol %d does not support reading emails", data.Proto)
	}
}

// AutoLoginSender 创建邮件发送客户端
func AutoLoginSender(data LoginParams) (EmailSender, error) {
	switch data.Proto {
	case SMTP:
		client, err := AutoLogin(data)
		if err != nil {
			return nil, err
		}
		return client.(EmailSender), nil
	default:
		return nil, fmt.Errorf("protocol %d does not support sending emails", data.Proto)
	}
}
