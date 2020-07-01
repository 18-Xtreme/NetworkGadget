package server

import (
	"NetworkGadget/src/main/model"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

var (
	lock                                 = sync.Mutex{}
	notice                   net.Conn    = nil
	noticeStartForward                   = make(chan int)
	waitForwardConnectionMap             = make(map[string]*ForwardModel)
	config                   *tls.Config = nil
)

func configServerTLS() {
	path := "certs/"
	cert, err := tls.LoadX509KeyPair(path+"server.pem", path+"server.key")
	if err != nil {
		log.Printf("加载服务端tls证书错误: %s\n", err.Error())
		return
	}
	certBytes, err := ioutil.ReadFile(path + "client.pem")
	if err != nil {
		log.Printf("[-] 读取cert.pem错误\n")
		return
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		log.Printf("[-] 解析tls加密错误\n")
		return
	}
	config = &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCertPool,
	}
}

func dealingWithClientConnection(base *model.ConfigBase) {
	var tcpListener net.Listener
	var err error

	if base.UseTLS {
		tcpListener, err = tls.Listen("tcp", base.SrcAddr, config)
	} else {
		// 监听来自客户端数据
		tcpListener, err = net.Listen("tcp", base.SrcAddr)
	}

	if err != nil {
		log.Printf("[-] 监听%d端口错误: %s\n", base.SrcPort, err.Error())
		return
	}
	defer tcpListener.Close()
	log.Printf("[+] 监听客户端连接%d端口.\n", base.SrcPort)

	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			log.Printf("[-] 接收来自%s数据错误: %s\n", tcpConn.RemoteAddr().String(), err.Error())
			continue
		}

		fmt.Println()
		log.Printf("[+] %s正在接入%d端口\n", tcpConn.RemoteAddr().String(), base.SrcPort)
		saveForForwardingConnection(tcpConn)
		informClientToCreateConnection("create\r")
	}
}

func saveForForwardingConnection(accept net.Conn) {
	lock.Lock()
	defer lock.Unlock()
	now := time.Now().UnixNano()
	// 时间戳+连接
	waitForwardConnectionMap[strconv.FormatInt(now, 10)] = &ForwardModel{accept, time.Now().Unix(), nil}
}

func informClientToCreateConnection(msg string) {
	log.Printf("[*] 正在通知客户端创建连接\n")
	if notice != nil {
		_, err := notice.Write([]byte(msg))
		if err != nil {
			log.Printf("[-] 发送信息错误: %s\n", err.Error())
		}
	} else {
		log.Printf("[-] 没有客户端连接,无法发送消息\n")
	}
}

func forwardingClientConnection(base *model.ConfigBase) {
	var tcpListener net.Listener
	var err error

	if base.UseTLS {
		tcpListener, err = tls.Listen("tcp", base.DstAddr, config)
	} else {
		// 监听客户端待转发数据
		tcpListener, err = net.Listen("tcp", base.DstAddr)
	}

	if err != nil {
		log.Printf("[-] 监听%d端口错误: %s\n", base.DstPort, err.Error())
		return
	}
	defer tcpListener.Close()

	log.Printf("[+] 监听转发%d端口.\n", base.DstPort)

	for {
		tcpConn, err := tcpListener.Accept()
		if err != nil {
			log.Printf("[-] 接收客户端连接错误: %s\n", err.Error())
			continue
		}
		log.Printf("[+] %s正在接入%d端口\n", tcpConn.RemoteAddr().String(), base.DstPort)

		if notice == nil {
			// 维持连接，用于通知客户端
			buf := make([]byte, 5)
			n, _ := tcpConn.Read(buf)
			if n == 5 && string(buf) == "start" {
				go handleClientNotice(tcpConn)
				continue
			}
		}

		// 配置转发通道
		configureForwardTunnel(tcpConn)
	}
}

func handleClientNotice(accept net.Conn) {
	log.Printf("[+] 客户端接入通知中心: %s\n", accept.RemoteAddr().String())
	if notice != nil {
		log.Printf("[!] 缓存通知连接已存在\n")
		accept.Close()
	} else {
		notice = accept
	}

	// 保持连接
	go keepAlive(accept)
}

func keepAlive(conn net.Conn) {
	for {
		// 每隔2秒发送一次数据
		_, err := conn.Write([]byte("keep\r"))
		if err != nil {
			log.Printf("[!] 客户端退出通知中心: %s\n", conn.RemoteAddr().String())
			notice = nil
			return
		}
		time.Sleep(time.Second * 2)
	}
}

func configureForwardTunnel(tunnel net.Conn) {
	lock.Lock()
	used := false
	// 遍历待转发连接map
	for _, connMatch := range waitForwardConnectionMap {
		// 将未设置转发通道的连接存入当前连接
		if connMatch.tunnel == nil && connMatch.accept != nil {
			connMatch.tunnel = tunnel
			used = true
			break
		}
	}

	if !used {
		log.Printf("[+] 已存在连接数: %d\n", len(waitForwardConnectionMap))
		_ = tunnel.Close()
		log.Printf("[+] 关闭多余的连接.\n")
	}
	lock.Unlock()
	// 通知开始转发
	noticeStartForward <- 1
}

func startTCPForward() {
	for {
		select {
		// 等待响应
		case <-noticeStartForward:
			lock.Lock()
			// 遍历待转发连接map
			for key, connMatch := range waitForwardConnectionMap {
				// 判断是否有对应的转发连接
				if connMatch.tunnel != nil && connMatch.accept != nil {
					log.Printf("[*] 创建TCP转发连接\n")
					go connectCopy(connMatch.accept, connMatch.tunnel)
					delete(waitForwardConnectionMap, key)
				}
			}
			lock.Unlock()
		}
	}
}

func connectCopy(accept, tunnel net.Conn) {
	copySwap := func(local, remote net.Conn) {
		defer local.Close()
		defer remote.Close()

		// 流拷贝
		_, err := io.Copy(local, remote)
		if err != nil {
			log.Printf("[-] 流拷贝错误: %s\n", err.Error())
			return
		}

		log.Printf("[+] 客户端连接转发完毕\n")
	}

	go copySwap(tunnel, accept)
	go copySwap(accept, tunnel)
}

func releaseTimeoutConnection() {
	for {
		lock.Lock()
		// 遍历转发连接map
		for key, connMatch := range waitForwardConnectionMap {
			// 获取没有设置转发连接的数据
			if connMatch.tunnel == nil && connMatch.accept != nil {
				// 判断是否超过5秒
				if time.Now().Unix()-connMatch.acceptAddTime > 5 {
					log.Printf("[!] 释放连接超时.\n")
					err := connMatch.accept.Close()
					if err != nil {
						log.Printf("[-] 释放连接错误: %s\n", err.Error())
					}
					// map中移除
					delete(waitForwardConnectionMap, key)
				}
			}
		}
		lock.Unlock()
		time.Sleep(time.Second * 5)
	}
}

func MainServer(base *model.ConfigBase) {
	if base.UseTLS {
		configServerTLS()
		if config == nil {
			log.Printf("[-] 配置服务端tls密钥错误.\n")
			return
		}
	}

	// 监听服务端口
	go dealingWithClientConnection(base)
	// 监听转发端口
	go forwardingClientConnection(base)
	// 释放超时连接
	go releaseTimeoutConnection()
	// tcp 转发
	startTCPForward()
}
