package client

import (
	"NetworkGadget/src/main/model"
	"log"
	"net"
	"sync"
	"time"
)

func P2PUdpClient() {
	// 本地监听地址
	local := &net.UDPAddr{IP: net.IPv4zero, Port: 51006}
	// 发送地址
	source := &net.UDPAddr{IP: net.IPv4zero, Port: 51000}
	// 服务器地址
	server := &net.UDPAddr{IP: net.ParseIP("192.168.3.123"), Port: 51006}
	buf := make([]byte, 4096)
	oob := make([]byte, 4096)

	conn, err := net.DialUDP("udp", source, server)
	if err != nil {
		log.Println(err)
		return
	}
	// 向服务器表示为客户端
	_, _ = conn.Write([]byte("client"))
	log.Println("[*] 等待服务器返回数据")

	// 读取点对点地址
	n, _, _ := conn.ReadFromUDP(buf)
	target, _ := net.ResolveUDPAddr("udp", string(buf[:n]))
	log.Println("[*] 目标UDP地址：", target)
	_ = conn.Close()

	go handleClientP2P(local, source, target, buf, oob, n)
}

func handleClientP2P(local, source, target *net.UDPAddr, buf, oob []byte, n int) {
	// p2p开始连接
	conn, err := net.DialUDP("udp", source, target)
	if err != nil {
		log.Println(target, err)
		return
	}

	// 玄学延迟1秒1
	time.Sleep(time.Second * 1)
	log.Println("玄学等待1秒 ^_^")

	_, err = conn.Write([]byte("握手包"))
	if err != nil {
		log.Println("发送握手包错误：", err)
		return
	}

	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Println("读取握手包错误：", err)
		return
	}
	log.Println("[^_^] 接收到：", string(buf[:n]))

	listen, err := net.ListenUDP("udp", local)
	if err != nil {
		log.Println("[-] 监听", local.Port, "端口错误：", err)
		return
	}
	log.Println("[*] 监听", local.Port, "端口成功")
	lock := new(sync.RWMutex)
	connections := make(map[string]*model.ConnectionUdp)

	for {
		n, _, _, addr, err := listen.ReadMsgUDP(buf, oob)
		if err != nil {
			log.Println("[-] 接收本地监听端口的数据错误：", err)
			return
		}

		go swapUdpData(lock, connections, addr, target, listen, conn, buf, n)
	}
}

func swapUdpData(lock *sync.RWMutex, connections map[string]*model.ConnectionUdp, addr, target *net.UDPAddr, listen, conn *net.UDPConn, buf []byte, n int) {
	lock.Lock()
	connection, found := connections[addr.String()]
	if !found {
		connection := new(model.ConnectionUdp)
		connection.Name = addr.String()
		connection.WaitChan = make(chan bool)
		connection.Udp = nil
		connections[addr.String()] = connection
	}
	lock.Unlock()
	if !found {
		lock.Lock()
		connections[addr.String()].Udp = conn
		close(connections[addr.String()].WaitChan)
		lock.Unlock()

		_, _, _ = conn.WriteMsgUDP(buf[:n], nil, nil)
		log.Println("[*] 首次UDP数据交互，地址：", target)

		for {
			buff := make([]byte, 4096)
			buffOob := make([]byte, 4096)
			length, _, _, _, err := conn.ReadMsgUDP(buff, buffOob)
			if err != nil {
				log.Println(err)
				return
			}

			n, _, err = listen.WriteMsgUDP(buff[:length], nil, addr)
			if err != nil {
				log.Println(err)
				return
			}
		}
	}

	// 此处需要注意，监听到的数据是连续的，但都是同一个连接，所以得在原来的基础继续发送
	// 切记不能开辟新新连接
	<-connection.WaitChan

	_, _, _ = connection.Udp.WriteMsgUDP(buf[:n], nil, nil)
	log.Println("[*] 持续发送交互数据，地址：", target)
}

func P2PUdpForward() {
	// 转发目的地址
	forwarder := &net.UDPAddr{IP: net.ParseIP("192.168.3.215"), Port: 5900}
	// 本地通讯地址
	source := &net.UDPAddr{IP: net.IPv4zero, Port: 51001}
	// 服务器地址
	server := &net.UDPAddr{IP: net.ParseIP("192.168.3.123"), Port: 51006}
	buf := make([]byte, 4096)
	buf1 := make([]byte, 4096)

	conn, err := net.DialUDP("udp", source, server)
	if err != nil {
		log.Println(err)
		return
	}
	// 向服务器表示为转发端
	_, _ = conn.Write([]byte("forward"))

	// 读取点对点地址
	n, _, _ := conn.ReadFromUDP(buf)
	target, _ := net.ResolveUDPAddr("udp", string(buf[:n]))
	log.Println("[*] 目标UDP地址：", target)
	_ = conn.Close()

	conn, err = net.DialUDP("udp", source, target)
	if err != nil {
		log.Println(target, err)
		return
	}

	// 玄学延迟1秒
	time.Sleep(time.Second * 1)
	log.Println("[*] 玄学等待1秒 ^_^")

	_, err = conn.Write([]byte("握手包"))
	if err != nil {
		log.Println("[-] 发送握手错误:", err)
		return
	}
	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Println("[-] 读取握手包错误：", err)
		return
	}

	log.Println("[^_^] 接收到：", string(buf[:n]))
	lock := new(sync.RWMutex)
	connections := make(map[string]*model.ConnectionUdp)

	for {
		n, _, _, addr, err := conn.ReadMsgUDP(buf, buf1)
		if err != nil {
			log.Println("[-] 读取来自", target, "数据包错误：", err)
			return
		}

		go handleForwardP2P(lock, connections, addr, forwarder, conn, buf, n)
	}
}

func handleForwardP2P(lock *sync.RWMutex, connections map[string]*model.ConnectionUdp, addr, forwarder *net.UDPAddr, conn *net.UDPConn, buf []byte, n int) {
	go func() {
		lock.Lock()
		connection, found := connections[addr.String()]
		if !found {
			connection := new(model.ConnectionUdp)
			connection.Name = addr.String()
			connection.WaitChan = make(chan bool)
			connection.Udp = nil
			connections[addr.String()] = connection
		}
		lock.Unlock()

		if !found {
			udpConn, err := net.DialUDP("udp", nil, forwarder)
			if err != nil {
				log.Println("[-] 连接到", forwarder, "错误：", err)
				return
			}
			log.Println("[*] 成功连接到：", forwarder)
			lock.Lock()
			connections[addr.String()].Udp = udpConn
			close(connections[addr.String()].WaitChan)
			lock.Unlock()

			_, _, _ = udpConn.WriteMsgUDP(buf[:n], nil, nil)
			log.Println("[*] 首次UDP数据交互，地址：", forwarder)

			for {
				buffer := make([]byte, 4096)
				bufferOob := make([]byte, 4096)
				length, _, _, _, err := udpConn.ReadMsgUDP(buffer, bufferOob)

				// 此处似乎有坑
				// _, _, _ = conn.WriteMsgUDP(buffer[:length], nil, addr)
				_, err = conn.Write(buffer[:length])
				if err != nil {
					log.Println(err)
					return
				}
			}
		}

		<-connection.WaitChan
		_, _, _ = connection.Udp.WriteMsgUDP(buf[:n], nil, nil)
		log.Println("[*] 持续发送交互数据，地址：", forwarder)
	}()
}
