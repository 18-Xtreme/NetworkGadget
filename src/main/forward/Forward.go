package forward

import (
	"NetworkGadget/src/main/client"
	"NetworkGadget/src/main/model"
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

var config *tls.Config = nil

func ListenPortToForwardConnect(base *model.ConfigBase, index string, proxy bool) {
	var listen net.Listener
	var err error

	if base.UseTLS && index != "2" {
		config = client.ConfigClientTLS()
		if config == nil {
			log.Printf("[-] 配置客户端tls密钥错误.\n")
			return
		}
		listen, err = tls.Listen("tcp", base.SrcAddr, config)
	} else {
		listen, err = net.Listen("tcp", base.SrcAddr)
	}

	if err != nil {
		log.Printf("[x] 监听端口%d错误:%s\n", base.SrcPort, err.Error())
		return
	}
	defer listen.Close()
	log.Printf("[*] 监听端口%d成功.\n", base.SrcPort)

	// 连接socks5代理时
	if proxy {
		// 读文件
		list := readProxyNodeConfigFile()
		if list == nil {
			log.Printf("[-] 读取代理节点文件错误,至少需要一个代理地址\n")
			return
		}
		addr := strings.Split(list[0], " ")
		// 设置一级代理服务器地址
		base.DstPort, err = strconv.Atoi(strings.ReplaceAll(addr[1], "\r\n", ""))
		base.DstAddr = fmt.Sprintf("%s:%d", addr[0], base.DstPort)

		// 移除一级代理服务器信息
		list = append(list[:0], list[1:]...)
		content := strings.Join(list, ",")
		content = strings.ReplaceAll(content, "\r\n", "")
		log.Printf("[+] 读取节点列表文件: %s\n", content)
		// 配置协议
		size := utils.Int16ToBytes(int16(len(content)))
		written := bytes.Join([][]byte{[]byte(model.ProtocolHeader), size, []byte(content)}, []byte(""))
		// 得到代理服务器连接，发送余剩节点数据
		netForward, err := getRemoteConnect(base, index)
		if err != nil {
			log.Printf("[-] 连接到%s错误: %s\n", base.DstAddr, err.Error())
			return
		}
		_, err = netForward.Write(written)
		if err != nil {
			log.Printf("[-] 发送节点列表信息错误: %s\n", err.Error())
		} else {
			log.Printf("[+] 发送节点列表信息成功\n")
		}
	}

	for {
		accept, err := listen.Accept()
		if err != nil {
			log.Printf("[x] 接收%s数据错误.\n", accept.RemoteAddr().String())
			continue
		}

		go handleForward(accept, base, index)
	}
}

func handleForward(accept net.Conn, base *model.ConfigBase, index string) {
	defer accept.Close()

	var netForward net.Conn
	var err error
	netForward, err = getRemoteConnect(base, index)

	if err != nil {
		log.Printf("[x] 连接端口%d错误:%s\n", base.DstPort, err.Error())
		return
	}
	defer netForward.Close()

	wg := new(sync.WaitGroup)
	wg.Add(2)

	if base.UseTLS {
		go func() {
			defer wg.Done()
			_, _ = client.OtherWayForward(netForward, accept)
		}()

		go func() {
			defer wg.Done()
			_, _ = client.OtherWayForward(accept, netForward)
		}()

	} else {
		go NormalForward(netForward, accept, wg)
		go NormalForward(accept, netForward, wg)
	}

	wg.Wait()
}

func NormalForward(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("[x] 流量转发错误:%s\n", err.Error())
	}
	// log.Printf("%s -> %s  %d/bytes", dst.RemoteAddr(), src.RemoteAddr(), written)
}

func getRemoteConnect(base *model.ConfigBase, index string) (netForward net.Conn, err error) {
	if base.UseTLS && index != "1" {
		if index == "2" {
			config = client.ConfigClientTLS()
			if config == nil {
				log.Printf("[-] 配置客户端tls密钥错误.\n")
				return
			}
		}
		netForward, err = tls.Dial("tcp", base.DstAddr, config)
	} else {
		netForward, err = net.Dial("tcp", base.DstAddr)
	}

	return netForward, err
}

func readProxyNodeConfigFile() (list []string) {
	_ = utils.ReadLine("proxy_node", func(bytes []byte) {
		line := string(bytes)
		if line != "\r\n" && line != "" {
			list = append(list, line)
		}
	})

	return list
}
