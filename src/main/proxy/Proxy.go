package proxy

import (
	"NetworkGadget/src/main/client"
	"NetworkGadget/src/main/forward"
	"NetworkGadget/src/main/model"
	"NetworkGadget/src/main/protocol"
	"NetworkGadget/src/main/server"
	"NetworkGadget/src/main/utils"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

var (
	nextNode   *model.NextProxyNode = nil
	config     *tls.Config          = nil
	exportNode                      = false
	sendNode                        = false
)

func StartProxy(base *model.ConfigBase) {
	var listen net.Listener
	var err error

	if base.UseTLS {
		config = server.ConfigServerTLS()
		if config == nil {
			return
		}
		listen, err = tls.Listen("tcp", base.SrcAddr, config)
	} else {
		listen, err = net.Listen("tcp", base.SrcAddr)
	}

	if err != nil {
		log.Printf("[-] 监听代理端口错误: %s\n", err.Error())
	}

	log.Printf("[+] 监听代理%d端口\n", base.SrcPort)

	for {
		accept, err := listen.Accept()
		if err != nil {
			log.Printf("[-] 接收数据错误: %s\n", err.Error())
			continue
		}

		go handleProxyService(accept, base)
	}
}

func handleProxyService(accept net.Conn, base *model.ConfigBase) {
	defer accept.Close()
	buf := make([]byte, 4)
	var list []string

	if nextNode == nil {
		accept.Read(buf)
		if string(buf) == "size" {
			log.Printf("[+] 开始接收节点列表\n")
			list = handleReceivingDataProtocol(accept, buf)
		} else {
			log.Printf("[!] 过滤未发送节点信息的连接\n")
			return
		}

		if exportNode {
			log.Printf("[+] 接收完毕，当前为出口节点\n")
		} else {
			log.Printf("[+] 接收完毕，当前为中转节点\n")
		}
	}

	// 长度0，即为出口节点
	if exportNode {
		handleExportNodeConnection(accept)
	} else {
		handleConnectionForward(accept, list, base)
	}

}

func handleReceivingDataProtocol(accept net.Conn, buf []byte) (list []string) {
	nextNode = new(model.NextProxyNode)
	// 读取数据长度
	accept.Read(buf)
	// []byte转int16
	size := utils.BytesToInt16(buf)
	// 最后一个节点
	if size == 0 {
		exportNode = true
		return make([]string, 0)
	}
	// 读取下个节点信息
	buf = make([]byte, size)
	accept.Read(buf)
	content := string(buf)
	log.Printf("[+] 节点列表: %s\n", content)
	// 切割存入结构体
	list = strings.Split(content, ",")
	addr := strings.Split(list[0], " ")
	nextNode.NodePort, _ = strconv.Atoi(addr[1])
	nextNode.NodeAddr = fmt.Sprintf("%s:%d", addr[0], nextNode.NodePort)
	log.Printf("[+] 下个节点信息:%v", nextNode)
	// 移除当前节点
	list = append(list[:0], list[1:]...)

	return list
}

func handleConnectionForward(accept net.Conn, list []string, base *model.ConfigBase) {
	var netForward net.Conn
	var err error

	if base.UseTLS {
		config = client.ConfigClientTLS()
		if config == nil {
			nextNode = nil
			return
		}
		netForward, err = tls.Dial("tcp", nextNode.NodeAddr, config)
	} else {
		netForward, err = net.Dial("tcp", nextNode.NodeAddr)
	}

	if err != nil {
		log.Printf("[-] 连接%s节点错误: %s\n", nextNode.NodeAddr, err.Error())
		nextNode = nil
		return
	}

	// 传输余剩节点信息
	if !sendNode {
		var content string
		size := utils.Int16ToBytes(int16(0))

		if len(list) > 0 {
			content = strings.Join(list, ",")
			size = utils.Int16ToBytes(int16(len(content)))
		}

		written := bytes.Join([][]byte{[]byte("size"), size, []byte(content)}, []byte(""))
		_, err = netForward.Write(written)
		if err != nil {
			log.Printf("[-] 传输余剩节点信息错误\n")
			nextNode = nil
			return
		} else {
			log.Printf("[+] 传输余剩节点信息到%s成功\n", nextNode.NodeAddr)
		}

		sendNode = true
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	if base.UseTLS {
		go func() {
			defer wg.Done()
			client.OtherWayForward(netForward, accept)
		}()

		go func() {
			defer wg.Done()
			client.OtherWayForward(accept, netForward)
		}()

	} else {
		go forward.NormalForward(netForward, accept, wg)
		go forward.NormalForward(accept, netForward, wg)
	}

	wg.Wait()
}

func handleExportNodeConnection(accept net.Conn) {
	defer accept.Close()
	buf := make([]byte, 1024)
	n, err := accept.Read(buf)
	if err != nil && err != io.EOF {
		log.Printf("[-] 读取数据流错误%s\n", err.Error())
		return
	}

	// 当为socks5协议
	if buf[0] == 0x05 {
		// 0x05代表socks5协议，0x00代表服务器不需要验证
		accept.Write([]byte{0x05, 0x00})
		n, err = accept.Read(buf)
		host, port := protocol.ParsingSocks5GetIpAndPort(buf, n)
		// log.Printf("connect to %s:%s", host, port)

		target, err := net.Dial("tcp", net.JoinHostPort(host, port))

		if err != nil {
			log.Println(err)
			return
		}
		defer target.Close()
		//响应客户端连接成功
		accept.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		wg := new(sync.WaitGroup)
		wg.Add(2)

		go forward.NormalForward(target, accept, wg)
		go forward.NormalForward(accept, target, wg)

		wg.Wait()
	}
}
