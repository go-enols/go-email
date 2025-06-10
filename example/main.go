package main

import (
	"os"

	"github.com/go-enols/go-log"

	"github.com/go-enols/go-email"
)

func main() {
	// 连接到IMAP服务器
	client, err := email.AutoLogin(email.LoginParams{
		Host:  "pop.qq.com",
		Port:  995,
		User:  "2575169674@qq.com",
		Pwd:   "your password",
		Proto: email.POP3,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	data, err := client.GetEmail(1) // 获取最新10封邮件
	if err != nil {
		return
	}

	os.WriteFile("./email.html", []byte(data[len(data)-1].HTMLBody), os.ModeAppend)

}
