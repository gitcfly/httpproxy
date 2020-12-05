package main

import (
	"fmt"
	"github.com/gitcfly/httpproxy/ioutils"
	"github.com/sirupsen/logrus"
	"net"
)

func main() {
	httpServer, _ := net.Listen("tcp", ":1234")
	for {
		conn, _ := httpServer.Accept()
		fmt.Println("收到数据请求：")
		data, _ := ioutils.ReadHttp(conn)
		fmt.Println(string(data))
		local, err := net.Dial("tcp", "127.0.0.1:8080")
		if err != nil {
			logrus.Error(err)
			return
		}
		if nl, err := local.Write(data); nl != len(data) || err != nil {
			logrus.Infof("写请求数据失败，总计长度 %v，写入长度：%v, err=%v,", len(data), nl, err)
		}
		ioutils.TransHttp(conn, local)
		local.Close()
		conn.Close()
		fmt.Println("响应完成")
	}
}
