package main

import (
	"bufio"
	"bytes"
	"container/list"
	"net"
	"sync"
	"time"

	"github.com/gitcfly/httpproxy/ioutils"
	"github.com/gitcfly/httpproxy/log"
	"github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
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

var TcpRecords = list.New()

type TcpRecord struct {
	clientKey   string
	sgnConn     net.Conn
	tcpListener net.Listener
}

var privateKey2port = map[string]string{
	"client_pkey_1": ":8888",
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
	go HeartBreakCheck()
	for {
		signalConn, err := signalListener.Accept()
		if err != nil {
			logrus.Error(err)
			return
		}
		go SignalClient(signalConn)
	}
}

func SignalClient(signalConn net.Conn) {
	privateBytes, _ := bufio.NewReader(signalConn).ReadBytes('\n')
	privateKey := string(bytes.TrimRight(privateBytes, "\n"))
	if listener := privateKey2Server[privateKey]; listener != nil {
		if err := listener.Close(); err != nil {
			logrus.Error(err)
		}
		delete(privateKey2Server, privateKey)
	}
	addr, ok := privateKey2port[privateKey]
	if !ok || addr == "" {
		logrus.Error("未配置客户端口，key=%v", privateKey)
		return
	}
	tcpListener, err := net.Listen("tcp", addr) //外部请求地址
	if err != nil {
		logrus.Error(err)
		return
	}
	TcpRecords.PushBack(&TcpRecord{
		clientKey:   privateKey,
		sgnConn:     signalConn,
		tcpListener: tcpListener},
	)
	privateKey2Server[privateKey] = tcpListener
	TcpServer(tcpListener, signalConn)
}

func HeartBreakCheck() {
	tmpData := make([]byte, 1)
	for range time.Tick(5 * time.Second) {
		var next *list.Element
		for e := TcpRecords.Front(); e != nil; e = next {
			next = e.Next()
			tcpRecord := e.Value.(*TcpRecord)
			if _, err := tcpRecord.sgnConn.Read(tmpData); err != nil {
				logrus.Infof("client_key=%v,客户端代理连接超时，服务端主动关闭连接以及tcp服务,err=%v", tcpRecord.clientKey, err)
				if err := tcpRecord.sgnConn.Close(); err != nil {
					logrus.Error(err)
				}
				if err := tcpRecord.tcpListener.Close(); err != nil {
					logrus.Error(err)
				}
				Request2Conn.Delete(tcpRecord.clientKey)
				TcpRecords.Remove(e)
			}
		}
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
		go RedirectConn(reqConn, requestId)
	}
}

func RedirectConn(reqConn net.Conn, requestId string) {
	defer func() {
		Request2Conn.Delete(requestId)
		logrus.Infof("请求处理结束，requestId=%v", requestId)
		if err := recover(); err != nil {
			logrus.Error(err)
		}
	}()
	for {
		if tpConn, ok := Request2Conn.Load(requestId); ok {
			proxyConn := tpConn.(net.Conn)
			go ioutils.CopyTcp(reqConn, proxyConn)
			ioutils.CopyTcp(proxyConn, reqConn)
			break
		}
	}
}
