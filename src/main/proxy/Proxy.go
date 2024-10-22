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
	headerBuf := make([]byte, 4)
	var list []string

	_, _ = accept.Read(headerBuf)
	if string(headerBuf) == model.ProtocolHeader {
		log.Printf("[+] 开始接收节点列表\n")
		list = handleReceivingDataProtocol(accept, base)

		if exportNode {
			log.Printf("[+] 接收完毕，当前为出口节点\n")
			return
		} else {
			log.Printf("[+] 接收完毕，当前为中转节点\n")
		}
		sendNode = false
	}

	// 长度0，即为出口节点
	if exportNode {
		handleExportNodeConnection(accept, headerBuf, base)
	} else {
		if nextNode == nil {
			log.Printf("[!] 未设置节点,过滤数据连接\n")
			return
		}
		handleConnectionForward(accept, headerBuf, list, base)
	}

}

func handleReceivingDataProtocol(accept net.Conn, base *model.ConfigBase) (list []string) {
	nextNode = new(model.NextProxyNode)
	buf := make([]byte, 4)
	// 读取数据长度
	_, _ = accept.Read(buf)
	// []byte转int16
	size := utils.BytesToInt16(buf)
	// 最后一个节点
	if size == 0 {
		exportNode = true
		return make([]string, 0)
	}
	// 读取下个节点信息
	buf = make([]byte, size)
	_, _ = accept.Read(buf)
	content := string(buf)
	log.Printf("[+] 节点列表: %s\n", content)
	// 切割存入结构体
	list = strings.Split(content, ",")
	for i, v := range list {
		addr := strings.Split(v, " ")
		nextNode.NodePort, _ = strconv.Atoi(addr[1])
		nextNode.NodeAddr = fmt.Sprintf("%s:%d", addr[0], nextNode.NodePort)
		n, err := getNextNodeConnect(base)
		if err != nil {
			log.Printf("[!] {%s}代理节点失效,切换节点中... \n", nextNode.NodeAddr)
			continue
		}
		log.Printf("[+] 下个节点信息:%v", nextNode)
		// 移除失效节点
		list = list[i:]
		_ = n.Close()
		break
	}
	return list
}

func handleConnectionForward(accept net.Conn, buf []byte, list []string, base *model.ConfigBase) {
	netForward, err := getNextNodeConnect(base)

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

		written := bytes.Join([][]byte{[]byte(model.ProtocolHeader), size, []byte(content)}, []byte(""))
		_, err = netForward.Write(written)
		if err != nil {
			log.Printf("[-] 传输余剩节点信息错误\n")
			nextNode = nil
			return
		} else {
			log.Printf("[+] 传输余剩节点信息到%s成功\n", nextNode.NodeAddr)
		}

		sendNode = true
	} else {
		_, _ = netForward.Write(buf)
	}

	wg := new(sync.WaitGroup)
	wg.Add(2)

	go forward.NormalForward(netForward, accept, wg)
	go forward.NormalForward(accept, netForward, wg)

	wg.Wait()
}

func handleExportNodeConnection(accept net.Conn, headerBuf []byte, base *model.ConfigBase) {
	defer accept.Close()

	// 当为socks5协议
	if headerBuf[0] == 0x05 {
		// 0x05代表socks5协议，0x00代表服务器不需要验证
		_, _ = accept.Write([]byte{0x05, 0x00})
		buf := make([]byte, 1024)
		n, err := accept.Read(buf)
		if err != nil && err != io.EOF {
			log.Printf("[-] 读取数据流错误%s\n", err.Error())
			return
		}
		host, port := protocol.ParsingSocks5GetIpAndPort(buf, n)
		log.Printf("connect to %s:%s", host, port)

		target, err := net.Dial("tcp", net.JoinHostPort(host, port))

		if err != nil {
			log.Println(err)
			return
		}
		defer target.Close()
		//响应客户端连接成功
		_, _ = accept.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

		wg := new(sync.WaitGroup)
		wg.Add(2)

		go forward.NormalForward(target, accept, wg)
		go forward.NormalForward(accept, target, wg)

		wg.Wait()
	}
}

func getNextNodeConnect(base *model.ConfigBase) (netForward net.Conn, err error) {
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

	return netForward, err
}
