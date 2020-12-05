package main

import (
	"bufio"
	"github.com/gitcfly/httpproxy/log"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		TimestampFormat:        "2006-01-02 15:04:05",
		FullTimestamp:          true,
		DisableLevelTruncation: true,
	})
	logrus.AddHook(log.NewContextHook())
}

var clientPraviteKey = "client_pkey_1\n"

// tcp内网端口代理,支持http协议，客户端实现
func main() {
	signalConn, err := net.Dial("tcp", "127.0.0.1:7777")
	if err != nil {
		logrus.Error(err)
		return
	}
	signalConn.Write([]byte(clientPraviteKey))
	for {
		requestId, err := bufio.NewReader(signalConn).ReadBytes('\n')
		if err != nil {
			logrus.Error(err)
			signalConn = RetrySignalConn()
			continue
		}
		logrus.Infof("读取到requestId=%v", string(requestId))
		go HandleTcpConn(string(requestId))
	}
}

func RetrySignalConn() net.Conn {
	for range time.Tick(2 * time.Second) {
		if signalConn, err := net.Dial("tcp", "127.0.0.1:7777"); err == nil {
			signalConn.Write([]byte(clientPraviteKey))
			return signalConn
		} else {
			logrus.Error(err)
		}
	}
	return nil
}

func HandleTcpConn(requestId string) {
	defer func() {
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	proxyConn, err := net.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		logrus.Error(err)
		return
	}
	proxyConn.Write([]byte(requestId))
	realConn, err := net.Dial("tcp", "127.0.0.1:8080")
	if err != nil {
		logrus.Error(err)
		return
	}
	go io.Copy(realConn, proxyConn)
	io.Copy(proxyConn, realConn)
}
