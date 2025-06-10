package email

import (
	"fmt"

	"github.com/emersion/go-imap/v2/imapclient"
)

type EmailIntface interface {
	Close() error
	GetEmail(opt ...any) ([]*ParsedMessage, error)
	ListMailboxes() ([]string, error)
	MonitEmail(opt ...any) ([]*ParsedMessage, error)
}

// createIMAPClient 创建IMAP客户端
func createIMAPClient(data LoginParams) (*ImapClient, error) {
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
func createPOP3Client(data LoginParams) (EmailIntface, error) {
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
func createSMTPClient(data LoginParams) (EmailIntface, error) {

	return nil, nil
}

func AutoLogin(data LoginParams) (EmailIntface, error) {
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
