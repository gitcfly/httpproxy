package ioutils

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
)

// 读取http数据流
func ReadHttp(conn net.Conn) ([]byte, error) {
	reader := bufio.NewReader(conn)
	var (
		data       []byte
		contentLen = -1
		endHeader  = false
		proto      = ""
		line       = 0
		chunked    = false
	)
	for { // 读取header
		bytes, err := reader.ReadBytes('\n')
		data = append(data, bytes...)
		line = line + 1
		if err != nil { //短连接读结束
			return data, err
		}
		tmpData := string(bytes)
		if line == 1 {
			idx := strings.Index(tmpData, " ")
			proto = tmpData[:idx]
		}
		if contentLen == -1 && strings.HasPrefix(tmpData, "Content-Length: ") {
			ctlenth := strings.Split(tmpData, ": ")[1]
			contentLen, _ = strconv.Atoi(strings.TrimSpace(ctlenth))
		}
		if !chunked && strings.HasPrefix(tmpData, "Transfer-Encoding: chunked") {
			chunked = true
		}
		if !endHeader && tmpData == "\r\n" {
			endHeader = true
		}
		if endHeader { //请求头读取结束
			break
		}
	}
	if contentLen == 0 {
		return data, nil
	}
	if contentLen > 0 {
		readLen := 0
		body := make([]byte, 2048)
		for {
			n, err := reader.Read(body)
			data = append(data, body[:n]...)
			if err != nil {
				logrus.Errorf("读取body 错误，contentLen=%v,readLen=%v,err=%v", contentLen, readLen, err)
				return data, err
			}
			readLen += n
			if readLen >= contentLen {
				break
			}
		}
		if readLen != contentLen {
			logrus.Errorf("读取body 错误，contentLen=%v,readLen=%v", contentLen, readLen)
		}
		return data, nil
	}
	if !chunked && strings.HasPrefix(proto, "HTTP") {
		return data, nil
	}
	if strings.HasPrefix(proto, "GET") {
		return data, nil
	}
	for {
		bytes, err := reader.ReadBytes('\n') // 数据长度
		data = append(data, bytes...)
		if err != nil {
			logrus.Errorf("读取body 错误，err=%v", err)
			return data, err
		}
		if string(bytes) != "0\r\n" {
			continue
		}
		data = append(data, []byte("\r\n")...)
		break
	}
	fmt.Println("读取数据完成。。。")
	return data, nil
}

// 转发http数据流
func TransHttp(dest net.Conn, conn net.Conn) error {
	reader := bufio.NewReader(conn)
	var (
		contentLen = -1
		endHeader  = false
		proto      = ""
		line       = 0
		chunked    = false
	)
	for { // 读取header
		bytes, err := reader.ReadBytes('\n')
		dest.Write(bytes)
		line = line + 1
		if err != nil { //短连接读结束
			return err
		}
		tmpData := string(bytes)
		if line == 1 {
			idx := strings.Index(tmpData, " ")
			proto = tmpData[:idx]
		}
		if contentLen == -1 && strings.HasPrefix(tmpData, "Content-Length: ") {
			ctlenth := strings.Split(tmpData, ": ")[1]
			contentLen, _ = strconv.Atoi(strings.TrimSpace(ctlenth))
		}
		if !chunked && strings.HasPrefix(tmpData, "Transfer-Encoding: chunked") {
			chunked = true
		}
		if !endHeader && tmpData == "\r\n" {
			endHeader = true
		}
		if endHeader { //请求头读取结束
			break
		}
	}
	if contentLen == 0 {
		return nil
	}
	if contentLen > 0 {
		readLen := 0
		body := make([]byte, 2048)
		var data []byte
		for {
			n, err := reader.Read(body)
			data = append(data, body[:n]...)
			if err != nil {
				logrus.Errorf("读取body 错误，contentLen=%v,readLen=%v,err=%v", contentLen, readLen, err)
				break
			}
			readLen += n
			if readLen >= contentLen {
				break
			}
		}
		dest.Write(data)
		if readLen != contentLen {
			logrus.Errorf("读取body 错误，contentLen=%v,readLen=%v", contentLen, readLen)
		}
		return nil
	}
	if !chunked && strings.HasPrefix(proto, "HTTP") {
		return nil
	}
	if strings.HasPrefix(proto, "GET") {
		return nil
	}
	for {
		bytes, err := reader.ReadBytes('\n') // 数据长度
		dest.Write(bytes)
		if err != nil {
			logrus.Errorf("读取body 错误，err=%v", err)
			return err
		}
		if string(bytes) != "0\r\n" {
			continue
		}
		dest.Write([]byte("\r\n"))
		break
	}
	fmt.Println("读取数据完成。。。")
	return nil
}
