package main

import (
	"fmt"
	"net"

	"github.com/gitcfly/httpproxy/config"
	"github.com/gitcfly/httpproxy/ioutils"
	"github.com/sirupsen/logrus"
)

var client2Server = map[string]net.Listener{}

func AcceptTcp() {
	tcpPort := config.Config.ServerTcpPort
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", tcpPort))
	if err != nil {
		logrus.Error(err)
		return
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		logrus.Error(err)
		return
	}
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			logrus.Errorf("listener.AcceptTCP() err,err=%v", err)
			continue
		}
		fmt.Println("收到连接请求。。。")
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			logrus.Errorf("读取客户端名称失败，error=%v", err)
		}
		clientKey := string(buf[:n])
		logrus.Infof("client_key=%v", clientKey)
		port := config.Config.ClientMapping[clientKey]
		if port == 0 {
			logrus.Errorf("client_key=%v，对应端口为空", clientKey)
			continue
		}
		if server := client2Server[clientKey]; server != nil {
			if err := server.Close(); err != nil {
				logrus.Errorf("server.Close() error,client_key=%v,error=%v", clientKey, err)
			}
			logrus.Errorf("client_key=%v，连接被重置", clientKey)
		}
		logrus.Infof("监听地址,%v", fmt.Sprintf(":%v", port))
		httpServer, err := net.Listen("tcp", fmt.Sprintf(":%v", port))
		if err != nil {
			logrus.Errorf("net.Listen error ,error=%v", err)
			return
		}
		logrus.Info("准备开启服务")
		go HttpServ(conn, httpServer)
		client2Server[clientKey] = httpServer
	}
}

func HttpServ(proxyConn net.Conn, listener net.Listener) {
	for {
		reqConn, err := listener.Accept()
		if err != nil {
			logrus.Infof("listener.Accept() error,error=%v", err)
			break
		}
		func(proxy net.Conn, client net.Conn) {
			defer func() {
				if err := recover(); err != nil {
					logrus.Errorf("io.Copy(dest, clent) panic,err=%v", err)
				}
			}()
			logrus.Infof("处理客户端http请求...")
			ioutils.TransHttp(proxy, client)
			ioutils.TransHttp(client, proxy)
			client.Close()
			logrus.Infof("请求处理完成")
		}(proxyConn, reqConn)
	}
}

func main() {
	AcceptTcp()
}
