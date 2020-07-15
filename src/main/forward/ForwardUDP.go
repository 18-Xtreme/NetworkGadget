package forward

import (
	"NetworkGadget/src/main/model"
	"log"
	"net"
	"sync"
)

const bufferSize = 4096

func UdpForward(base *model.ConfigBase) {
	src, _ := net.ResolveUDPAddr("udp", base.SrcAddr)
	dst, _ := net.ResolveUDPAddr("udp", base.DstAddr)
	lock := new(sync.RWMutex)
	connections := make(map[string]*model.ConnectionUdp)

	listen, err := net.ListenUDP("udp", src)
	if err != nil {
		log.Println("[-] 监听udp", base.SrcPort, "端口错误：", err)
		return
	}
	log.Println("[*] 监听", base.SrcPort, "端口成功")

	buf := make([]byte, bufferSize)
	oob := make([]byte, bufferSize)
	for {
		n, _, _, addr, err := listen.ReadMsgUDP(buf, oob)
		if err != nil {
			log.Println("[-] 读取待转发数据错误：", err)
			return
		}

		go handleUdpForward(listen, lock, connections, addr, dst, buf, n)
	}
}

func handleUdpForward(listen *net.UDPConn, lock *sync.RWMutex, connections map[string]*model.ConnectionUdp, addr, dst *net.UDPAddr, buf []byte, n int) {
	lock.Lock()
	conn, found := connections[addr.String()]
	if !found {
		log.Println("[*] 新的连接进入")
		connection := new(model.ConnectionUdp)
		connection.WaitChan = make(chan bool)
		connection.Name = addr.String()
		connection.Udp = nil
		connections[addr.String()] = connection
	}
	lock.Unlock()

	if !found {
		udpConn, err := net.DialUDP("udp", nil, dst)
		if err != nil {
			log.Println("[-] 连接", dst, "错误：", err)
			return
		}
		log.Println("[*] 成功连接到", dst)
		lock.Lock()
		connections[addr.String()].Udp = udpConn
		close(connections[addr.String()].WaitChan)
		lock.Unlock()

		_, _, err = udpConn.WriteMsgUDP(buf[:n], nil, nil)
		if err != nil {
			log.Println("[-] 发送转发数据包到", dst, "错误：", err)
		}

		for {
			buf := make([]byte, bufferSize)
			oob := make([]byte, bufferSize)
			n, _, _, _, err := udpConn.ReadMsgUDP(buf, oob)
			if err != nil {
				_ = udpConn.Close()
				log.Println("[-] 读取来自", dst, "数据错误：", err)
				return
			}

			_, _, err = listen.WriteMsgUDP(buf[:n], nil, addr)
			if err != nil {
				log.Println("[-] 转发数据包到本地错误：", err)
			}
		}
	}

	<-conn.WaitChan

	_, _, err := conn.Udp.WriteMsgUDP(buf[:n], nil, nil)
	if err != nil {
		log.Println("[-] 持续发送待转发数据包错误:", err)
	}
}
