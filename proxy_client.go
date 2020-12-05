package main

import (
	"fmt"
	"github.com/gitcfly/httpproxy/config"
	"github.com/gitcfly/httpproxy/ioutils"
	"github.com/sirupsen/logrus"
	"net"
	"time"
)

func main() {
	tcpServerAddr := fmt.Sprintf("%v:%v", config.Config.ServerHost, config.Config.ServerTcpPort)
	tcpConn, _ := net.Dial("tcp", tcpServerAddr)
	tcpConn.Write([]byte("client_key1"))
	for {
		req, err := ioutils.ReadHttp(tcpConn)
		if err != nil {
			logrus.Info("远程连接被关闭。。")
			for tcpConn, err = net.Dial("tcp", tcpServerAddr); err != nil; {
				logrus.Errorf("连接重试中。。，error=%v", err)
				time.Sleep(2 * time.Second)
			}
		}
		func() {
			defer func() {
				if err := recover(); err != nil {
					logrus.Errorf("请求处理失败，发生了panic,err=%v", err)
				}
			}()
			logrus.Info("开始播号。。")
			local, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%v", config.Config.LocHttpPort))
			if err != nil {
				logrus.Error(err)
				return
			}
			if nl, err := local.Write(req); nl != len(req) || err != nil {
				logrus.Infof("写请求数据失败，总计长度 %v，写入长度：%v, err=%v,", len(req), nl, err)
			}
			ioutils.TransHttp(tcpConn, local)
			logrus.Infof("响应数据完成")
		}()
	}
}
