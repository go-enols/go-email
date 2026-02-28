package main

import (
	"fmt"

	"github.com/go-enols/go-log"

	"github.com/go-enols/go-email"
)

func POP3() {
	// 连接到POP3服务器
	client, err := email.AutoLoginReader(email.LoginParams{
		Host:  "pop.qq.com",
		Port:  995,
		User:  "2575169674@qq.com",
		Pwd:   "you'r password",
		Proto: email.POP3,
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer client.Close()

	data, err := client.GetEmail(1) // 获取最新1封邮件
	if err != nil {
		log.Error(err)
		return
	}

	fmt.Println("POP3:", len(data))
}

func SMTP() {
	// 连接到SMTP服务器
	client, err := email.AutoLoginSender(email.LoginParams{
		Host:  "smtp.qq.com",
		Port:  587,
		User:  "2575169674@qq.com",
		Pwd:   "you'r password",
		Proto: email.SMTP,
	})
	if err != nil {
		log.Error(err)
		return
	}
	defer client.Close()

	// 发送邮件
	err = client.SendEmail(
		[]string{"2575169674@qq.com"}, // 收件人
		"测试邮件",                        // 主题
		"这是一封测试邮件",                    // 正文
		[]*email.Attachment{},         // 附件
	)
	if err != nil {
		log.Error(err)
		return
	}

	fmt.Println("SMTP: 邮件发送成功")
}

func IMAP() {
	// 连接到POP3服务器
	client, err := email.AutoLoginReader(email.LoginParams{
		Host:  "pop.qq.com",
		Port:  993,
		User:  "2575169674@qq.com",
		Pwd:   "you'r password",
		Proto: email.IMAP,
	})
	if err != nil {
		log.Error(err)
	}
	defer client.Close()

	data, err := client.GetEmail(1) // 获取最新1封邮件
	if err != nil {
		log.Error(err)
	}

	fmt.Println("IMAP:", len(data))
}
func main() {
	POP3()
	IMAP()
	SMTP()
}
