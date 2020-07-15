package server

import (
	"fmt"
	"log"
	"net"
	"time"
)

const bufferSize = 4096

func P2PUdpServer(port string) {
	server, _ := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", port))
	listen, err := net.ListenUDP("udp", server)
	if err != nil {
		log.Println("[-] 监听", port, "端口错误：", err)
		return
	}
	log.Println("[*] 监听", server.Port, "端口成功")

	var forwardAddr, clientAddr *net.UDPAddr
	buf := make([]byte, bufferSize)
	log.Println("[*] 等待连接中...")

	for {
		n, addr, err := listen.ReadFromUDP(buf)
		if err != nil {
			log.Println("[-] 读取来自", addr, "数据错误：", err)
			return
		}
		if string(buf[:n]) == "forward" && forwardAddr == nil {
			forwardAddr = addr
			log.Println("[*] 转发端接入")
		} else if string(buf[:n]) == "client" && clientAddr == nil {
			clientAddr = addr
			log.Println("[*] 客户端接入")
		}

		if forwardAddr != nil && clientAddr != nil {
			// 如果出口IP相同，直接nat转发，不经过外网
			if forwardAddr.IP.Equal(clientAddr.IP) {
				// TODO： 待补充
				log.Println("[!] 出口IP相同，NAT直接转发...(功能待完善)")
			}
			// TODO：缺少检测NAT类型，动态类型的提示NAT不支持

			_, writeClientErr := listen.WriteToUDP([]byte(forwardAddr.String()), clientAddr)
			_, writeFrowardErr := listen.WriteToUDP([]byte(clientAddr.String()), forwardAddr)
			if writeClientErr != nil || writeFrowardErr != nil {
				log.Println("[-] 传输点对点地址错误")
			} else {
				log.Println("[*] 传输点对点地址成功")
			}
			log.Println("[*] 倒计时5秒退出...")
			time.Sleep(time.Second * 5)
			return
		}

	}
}
