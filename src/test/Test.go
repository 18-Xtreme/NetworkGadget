package test

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"sync"
)

type ConnModel struct {
	LocalAddr  string
	LocalPort  int
	TargetAddr string
	TargetPort int
	ServerAddr string
	ServerPort int
}

func ListenForwardPort(connModel *ConnModel) {
	// 监听目标端口
	local, err := net.Listen("tcp", connModel.LocalAddr)
	if err != nil {
		log.Printf("[x] Unable to listen port: %d err: %s", connModel.LocalPort, err.Error())
	}
	defer local.Close()
	log.Printf("Listen %d port successful.", connModel.LocalPort)
	log.Printf("Ready to forward %d port.", connModel.TargetPort)

	for {
		// 阻塞接收
		localConn, err := local.Accept()
		if err != nil {
			log.Printf("[!] Unable to accept a request, error: %s\n", err.Error())
			continue
		}

		log.Println("[*] new connect:" + localConn.RemoteAddr().String())

		// 处理接收连接
		go handleReceiverConnection(connModel.TargetAddr, connModel.TargetPort, localConn)
	}
}

func ListenServerPort(connModel *ConnModel) {
	connModel.ServerAddr = fmt.Sprintf(":%d", connModel.ServerPort)

	server, err := net.Listen("tcp", connModel.ServerAddr)
	if err != nil {
		log.Printf("[x] Unable to listen server port %d, error: %s\n", connModel.ServerPort, err.Error())
	}
	defer server.Close()
	log.Printf("Server Listen %d port successful.", connModel.ServerPort)

	for {
		serverConn, err := server.Accept()
		if err != nil {
			log.Printf("[!] Unable to accept a request, error: %s\n", err.Error())
		}

		log.Println("[*] new connect:" + serverConn.RemoteAddr().String())

		// 处理接收连接
		go handleReceiverConnection(connModel.LocalAddr, connModel.LocalPort, serverConn)
	}

}

func handleReceiverConnection(targetAddr string, targetPort int, localConn net.Conn) {
	defer localConn.Close()

	// 创建tcp连接 30秒过期
	targetConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		log.Printf("[x] Unable to connect source port: %d, error: %s\n", targetPort, err.Error())
		return
	}
	defer targetConn.Close()

	// 信号量
	wg := new(sync.WaitGroup)
	wg.Add(2)

	// 流复制协程 s -> t
	go connectCopyAndSwap(targetConn, localConn, wg)

	// 流复制协程 t -> s
	go connectCopyAndSwap(localConn, targetConn, wg)

	/*go func() {
		defer wg.Done()
		written, err := OtherWayCopy(targetConn, sourceConn)
		if err != nil {
			log.Printf("send data err: %s", err.Error())
		}
		log.Printf("%d\n", written)
	}()

	go func() {
		defer wg.Done()
		written, err := OtherWayCopy(sourceConn, targetConn)
		if err != nil {
			log.Printf("send data err: %s", err.Error())
		}
		log.Printf("%d\n", written)
	}()*/

	wg.Wait()

}

func connectCopyAndSwap(sourceConn, targetConn net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	written, err := io.Copy(sourceConn, targetConn)
	if err != nil {
		log.Printf("[x] sockicopy connection error: %s", err.Error())
	}
	log.Printf("[*] %s -> %s  %d/bytes", sourceConn.RemoteAddr().String(), targetConn.RemoteAddr().String(), written)

}

func otherWayForward(src, dst net.Conn) (written int64, err error) {
	size := 1024
	buf := make([]byte, size)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			log.Println(src.RemoteAddr().String(), "->", dst.RemoteAddr())
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

func TestMain() {
	path := "certs/"
	cert, err := tls.LoadX509KeyPair(path+"server.pem", path+"server.key")
	if err != nil {
		log.Println(err)
		return
	}
	certBytes, err := ioutil.ReadFile(path + "client.pem")
	if err != nil {
		panic("Unable to read cert.pem")
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("failed to parse root certificate")
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCertPool,
	}
	ln, err := tls.Listen("tcp", ":51001", config)
	if err != nil {
		log.Println(err)
		return
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		println(msg)
		n, err := conn.Write([]byte("world\n"))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}

func TestMain1() {
	path := "certs/"
	cert, err := tls.LoadX509KeyPair(path+"client.pem", path+"client.key")
	if err != nil {
		log.Println(err)
		return
	}
	certBytes, err := ioutil.ReadFile(path + "client.pem")
	if err != nil {
		panic("Unable to read cert.pem")
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("failed to parse root certificate")
	}
	conf := &tls.Config{
		RootCAs:            clientCertPool,
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", "127.0.0.1:51001", conf)
	if err != nil {
		log.Println(err)
		return
	}
	defer conn.Close()
	n, err := conn.Write([]byte("hello\n"))
	if err != nil {
		log.Println(n, err)
		return
	}
	buf := make([]byte, 100)
	n, err = conn.Read(buf)
	if err != nil {
		log.Println(n, err)
		return
	}
	println(string(buf[:n]))
}

func TestMain2() {
	listen, _ := net.Listen("tcp", ":5100")
	for {
		accept, _ := listen.Accept()
		buf := make([]byte, 512)
		n, _ := accept.Read(buf)
		log.Println(string(buf[:n]))
		accept.Write([]byte("world\n"))
		accept.Close()
	}
}

func TestMain3() {
	conn, _ := net.Dial("tcp", "192.168.3.123:3389")
	conn.Write([]byte("hello1\n"))
	conn.Close()
}
