package main

import (
	"bufio"
	"github.com/gitcfly/httpproxy/log"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
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

var privateKey2port = map[string]string{
	"client_pkey_1\n": ":8888",
}

var privateKey2Server = map[string]net.Listener{}

var Request2Conn = &sync.Map{}

// tcp内网端口代理,支持http协议，服务端实现
func main() {
	signalListener, err := net.Listen("tcp", ":7777")
	if err != nil {
		logrus.Error(err)
		return
	}
	go TcpConnPool()
	for {
		signalConn, err := signalListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		privateKey, _ := bufio.NewReader(signalConn).ReadBytes('\n')
		if listener := privateKey2Server[string(privateKey)]; listener != nil {
			if err := listener.Close(); err != nil {
				logrus.Error(err)
			}
			delete(privateKey2Server, string(privateKey))
		}
		addr := privateKey2port[string(privateKey)]
		tcpListener, err := net.Listen("tcp", addr) //外部请求地址
		if err != nil {
			logrus.Error(err)
			return
		}
		privateKey2Server[string(privateKey)] = tcpListener
		go TcpServer(tcpListener, signalConn)
	}
}

func TcpConnPool() {
	poolListener, err := net.Listen("tcp", ":9999") //内部连接池端口
	if err != nil {
		logrus.Error(err)
		return
	}
	for {
		tcpConn, err := poolListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		requestId, err := bufio.NewReader(tcpConn).ReadBytes('\n')
		Request2Conn.Store(string(requestId), tcpConn)
	}
}

func TcpServer(tcpListener net.Listener, signalConn net.Conn) {
	for {
		reqConn, err := tcpListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		requestId := uuid.NewV4().String() + "\n"
		logrus.Infof("写入requestId=%v", requestId)
		_, err = signalConn.Write([]byte(requestId)) //告知客户端需要主动发起连接请求
		if err != nil {
			logrus.Error(err)
			return
		}
		go TransConnReqest(reqConn, requestId)
	}
}

func TransConnReqest(reqConn net.Conn, requestId string) {
	defer func() {
		Request2Conn.Delete(requestId)
	}()
	for {
		if tpConn, ok := Request2Conn.Load(requestId); ok {
			proxyConn := tpConn.(net.Conn)
			go io.Copy(proxyConn, reqConn)
			io.Copy(reqConn, proxyConn)
			break
		}
	}
}
