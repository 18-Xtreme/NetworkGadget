package client

import (
	"NetworkGadget/src/main/model"
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
)

var config *tls.Config = nil

func ConfigClientTLS() *tls.Config {
	path := "certs/"
	cert, err := tls.LoadX509KeyPair(path+"client.pem", path+"client.key")
	if err != nil {
		log.Println(err)
		return nil
	}
	certBytes, err := ioutil.ReadFile(path + "client.pem")
	if err != nil {
		log.Printf("[-] 读取cert.pem错误\n")
		return nil
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		log.Printf("[-] 解析tls加密错误\n")
		return nil
	}
	config := &tls.Config{
		RootCAs:            clientCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	return config
}

func OtherWayForward(src, dst net.Conn) (written int64, err error) {
	size := 1024
	buf := make([]byte, size)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			// log.Println(src.RemoteAddr().String(), "->", dst.RemoteAddr())
			if nw > 0 {
				written += int64(nw)
			}

			if ew != nil {
				err = ew
				break
			}

			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}

		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}

	return written, err
}

func waitingForServerNotice(base *model.ConfigBase) {
	/*var tcpAddr *net.TCPAddr
	// 连接服务端通知端口
	tcpAddr, _ = net.ResolveTCPAddr("tcp", base.DstAddr)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)*/
	conn, err := tls.Dial("tcp", base.DstAddr, config)
	if err != nil {
		log.Printf("[-] 接入服务端通知中心(%s)错误: %s\n", base.DstAddr, err.Error())
		return
	}

	_, err = conn.Write([]byte("start"))
	if err != nil {
		log.Printf("[-] 接入服务端通知中心错误: %s\n", err.Error())
		return
	}

	log.Printf("[+] 成功接入服务端通知中心\n")
	// 读取来自服务端数据
	reader := bufio.NewReader(conn)
	for {
		s, err := reader.ReadString('\r')
		if err != nil || err == io.EOF {
			break
		} else {
			s = s[:len(s)-1]
			// 区分不同的通知
			if s == "create" {
				// 创建新连接，且处理来服务器的连接
				go handleLocalForwardingConnection(base)
			}
			if s == "keep" {
				//log.Printf("[~] 连接保持中 ...")
			}
		}
	}

}

func handleLocalForwardingConnection(base *model.ConfigBase) {
	local := connectLocal(base)
	remote := connectRemote(base)
	// 判断是否接入远程连接
	if local != nil && remote != nil {
		// 流复制
		// connectCopy(local, remote)
		go OtherWayForward(remote, local)
		go OtherWayForward(local, remote)
	} else {
		if local != nil {
			err := local.Close()
			if err != nil {

			}
		}

		if remote != nil {
			err := remote.Close()
			if err != nil {

			}
		}
	}
}

func connectCopy(local, remote *net.TCPConn) {
	copySwap := func(local, remote *net.TCPConn) {
		defer local.Close()
		defer remote.Close()

		_, err := io.Copy(local, remote)
		if err != nil {
			log.Printf("[-] 流复制错误: %s\n", err.Error())
			return
		}
		log.Printf("[+] 服务端连接转发完毕\n")
	}

	go copySwap(remote, local)
	go copySwap(local, remote)

}

func connectLocal(base *model.ConfigBase) *net.TCPConn {
	var tcpAddr *net.TCPAddr
	// 连接本地端口
	tcpAddr, _ = net.ResolveTCPAddr("tcp", base.SrcAddr)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Printf("[-] 连接本地端口%d错误: %s\n", base.SrcPort, err.Error())
		return nil
	}

	fmt.Println()
	log.Printf("[+] 成功接入本地%d端口\n", base.SrcPort)
	return conn
}

func connectRemote(base *model.ConfigBase) *tls.Conn {
	/*var tcpAddr *net.TCPAddr
	// 连接远程转发端口
	tcpAddr, _ = net.ResolveTCPAddr("tcp", base.DstAddr)
	conn, err := net.DialTCP("tcp", nil, tcpAddr)*/
	conn, err := tls.Dial("tcp", base.DstAddr, config)
	if err != nil {
		log.Printf("[-] 连接服务端转发端口(%d)错误: %s\n", base.DstPort, err.Error())
		return nil
	}

	log.Printf("[+] 成功接入服务端转发端口\n")
	return conn
}

func MainClient(base *model.ConfigBase) {
	config = ConfigClientTLS()
	if config == nil {
		log.Printf("[-] 配置客户端tls密钥错误.\n")
		return
	}

	waitingForServerNotice(base)
}
