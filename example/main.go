package main

import (
	"os"

	"github.com/go-enols/go-log"

	"github.com/go-enols/go-email"
)

func main() {
	// 连接到IMAP服务器
	client, err := email.AutoLogin(email.LoginParams{
		Host:         "outlook.office365.com",
		Port:         993,
		User:         "xxxx@hotmail.com",
		Pwd:          "xxx",
		ClientId:     "xxxx-xxx-46bd-ae65-b683e7707cb0",
		RefreshToken: "your token",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	log.Debug(client.ListMailboxes()) // 列出所有邮件箱
	data, err := client.GetEmail(10)  // 获取最新10封邮件
	if err != nil {
		return
	}
	os.WriteFile("./email.html", []byte(data[len(data)-1].HTMLBody), os.ModeAppend)

}
