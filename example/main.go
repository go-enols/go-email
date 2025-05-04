package main

import (
	"time"

	"github.com/go-enols/go-log"

	"github.com/go-enols/go-email"
)

func main() {
	// 连接到IMAP服务器
	client, err := email.Connect("邮箱服务器节点", "你的账号", "你的密码")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	log.Debug(client.ListMailboxes())                // 列出所有邮件箱
	log.Debug(client.GetEmail(10))                   // 获取最新10封邮件
	log.Debug(client.MonitEmail(10, time.Second*60)) // 监控最新10封邮件

}
