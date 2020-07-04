package test

import (
	"NetworkGadget/src/main/protocol"
	"io"
	"log"
	"net"
	"sync"
)

func ProxyStart() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	l, err := net.Listen("tcp", ":5100")
	if err != nil {
		log.Panic(err)
	}

	for {
		client, err := l.Accept()
		if err != nil {
			log.Panic(err)
		}

		go handleClientRequest(client)
	}
}

func handleClientRequest(client net.Conn) {
	defer client.Close()
	buf := make([]byte, 1024)
	n, err := client.Read(buf)
	if err != nil && err != io.EOF {
		log.Panic(err)
		return
	}

	// 当为socks5协议
	if buf[0] == 0x05 {
		// 0x05代表socks5协议，0x00代表服务器不需要验证
		client.Write([]byte{0x05, 0x00})
		n, err = client.Read(buf)
		host, port := protocol.ParsingSocks5GetIpAndPort(buf, n)

		target, err := net.Dial("tcp", net.JoinHostPort(host, port))
		if err != nil {
			log.Println(err)
			return
		}
		defer target.Close()
		//响应客户端连接成功
		client.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		wg := new(sync.WaitGroup)
		wg.Add(2)

		go normalForward(target, client, wg)
		go normalForward(client, target, wg)

		wg.Wait()
	}
}

func normalForward(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	written, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("[x] 流量转发错误:%s\n", err.Error())
	}
	log.Printf("%s -> %s  %d/bytes", dst.RemoteAddr(), src.RemoteAddr(), written)
}
