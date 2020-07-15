package test

import (
	"log"
	"net"
	"sync"
	"time"
)

type connectionUDP struct {
	name      string
	udp       *net.UDPConn
	waitWrite chan bool
}

func ForwardUDP(localAddr, targetAddr string) {
	src, _ := net.ResolveUDPAddr("udp", localAddr)
	dst, _ := net.ResolveUDPAddr("udp", targetAddr)
	lock := new(sync.RWMutex)
	//connection := new(connectionUDP)
	connections := make(map[string]*connectionUDP)

	listen, err := net.ListenUDP("udp", src)
	if err != nil {
		log.Println("[-] listen local udp port err:", err)
		return
	}

	buf := make([]byte, bufferSize)
	oob := make([]byte, bufferSize)
	for {
		n, _, _, addr, err := listen.ReadMsgUDP(buf, oob)
		if err != nil {
			log.Println("[-] forward: failed to read, terminating:", err)
			return
		}

		go handleForward(listen, lock, connections, addr, dst, buf, n)
	}

}

func handleForward(listen *net.UDPConn, lock *sync.RWMutex, connections map[string]*connectionUDP, addr, dst *net.UDPAddr, buf []byte, n int) {
	lock.Lock()
	conn, found := connections[addr.String()]
	if !found {
		log.Println("[*] new connect come in.")
		connection := new(connectionUDP)
		connection.waitWrite = make(chan bool)
		connection.name = addr.String()
		connection.udp = nil
		connections[addr.String()] = connection
	}
	lock.Unlock()

	if !found {
		udpConn, err := net.DialUDP("udp", nil, dst)
		if err != nil {
			log.Println("[-] udp-forward: failed to dial:", err)
			return
		}
		log.Println("[*] connect to", dst, "success.")
		lock.Lock()
		connections[addr.String()].udp = udpConn
		close(connections[addr.String()].waitWrite)
		lock.Unlock()

		_, _, err = udpConn.WriteMsgUDP(buf[:n], nil, nil)
		if err != nil {
			log.Println("[-] udp-forward: error sending initial packet to client", err)
		}
		//log.Println("[*] write data to", dst, "success.")

		for {
			buf := make([]byte, bufferSize)
			oob := make([]byte, bufferSize)
			n, _, _, _, err := udpConn.ReadMsgUDP(buf, oob)
			if err != nil {
				_ = udpConn.Close()
				log.Println("[*] udp-forward: abnormal read, closing:", err)
				return
			}
			//log.Println("[*] read data from", dst, "success.")

			_, _, err = listen.WriteMsgUDP(buf[:n], nil, addr)
			if err != nil {
				log.Println("[*] udp-forward: error sending packet to client:", err)
			}
			//log.Println("[*] write data to", src, "success.")
		}
	}

	<-conn.waitWrite

	_, _, err := conn.udp.WriteMsgUDP(buf[:n], nil, nil)
	if err != nil {
		log.Println("udp-forward: error sending packet to server:", err)
	}
}

func ServerUDP() {
	server, _ := net.ResolveUDPAddr("udp", ":51006")
	listen, err := net.ListenUDP("udp", server)
	if err != nil {
		log.Println("[-] listen server port err:", err)
		return
	}
	log.Println("[*] listen server", server.Port, "port")

	var forwardAddr, clientAddr *net.UDPAddr
	buf := make([]byte, bufferSize)
	for {

		n, addr, err := listen.ReadFromUDP(buf)
		if err != nil {
			log.Println("[-] read data from", addr, "err:", err)
			return
		}
		if string(buf[:n]) == "forward" {
			forwardAddr = addr
			log.Println("[*] forwarder come in.")
		} else if string(buf[:n]) == "client" {
			clientAddr = addr
			log.Println("[*] client come in.")
		}

		if forwardAddr != nil && clientAddr != nil {
			_, writeClientErr := listen.WriteToUDP([]byte(forwardAddr.String()), clientAddr)
			_, writeFrowardErr := listen.WriteToUDP([]byte(clientAddr.String()), forwardAddr)
			if writeClientErr != nil || writeFrowardErr != nil {
				log.Println("[-] send data err.")
			} else {
				log.Println("[*] send data success, will exit server.")
			}
			return
		}

	}
}

func ClientConnectToServer() {
	local := &net.UDPAddr{IP: net.IPv4zero, Port: 51006}
	source := &net.UDPAddr{IP: net.IPv4zero, Port: 9981}
	server := &net.UDPAddr{IP: net.ParseIP("192.168.3.123"), Port: 51006}
	buf := make([]byte, 4096)
	buf1 := make([]byte, 4096)

	conn, err := net.DialUDP("udp", source, server)
	if err != nil {
		log.Println(err)
		return
	}
	_, _ = conn.Write([]byte("client"))

	n, _, _ := conn.ReadFromUDP(buf)
	target, _ := net.ResolveUDPAddr("udp", string(buf[:n]))
	log.Println("receive data:", target)
	_ = conn.Close()

	conn, err = net.DialUDP("udp", source, target)
	if err != nil {
		log.Println(target, err)
		return
	}

	time.Sleep(time.Second * 1)
	log.Println("wait one second to send.")

	_, err = conn.Write([]byte("handshake packet."))
	if err != nil {
		log.Println("send handshake err:", err)
		return
	}

	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Println("read data err:", err)
		return
	}

	log.Println("get handshake packet yes.", string(buf[:n]))

	listen, err := net.ListenUDP("udp", local)
	if err != nil {
		log.Println("listen 51006 port err:", err)
		return
	}
	log.Println("listen 51006 port success.")
	lock := new(sync.RWMutex)
	connections := make(map[string]*connectionUDP)

	for {
		n, _, _, addr, err := listen.ReadMsgUDP(buf, buf1)
		if err != nil {
			log.Println("receive from 51006 data err:", err)
			return
		}
		log.Println("receive data from 51006")

		go func() {
			lock.Lock()
			connection, found := connections[addr.String()]
			if !found {
				connection := new(connectionUDP)
				connection.name = addr.String()
				connection.waitWrite = make(chan bool)
				connection.udp = nil
				connections[addr.String()] = connection
			}
			lock.Unlock()
			if !found {
				lock.Lock()
				connections[addr.String()].udp = conn
				close(connections[addr.String()].waitWrite)
				lock.Unlock()

				_, _, _ = conn.WriteMsgUDP(buf[:n], nil, nil)
				log.Println("first write data to", target)

				for {
					buff := make([]byte, 4096)
					buff1 := make([]byte, 4096)
					length, _, _, _, err := conn.ReadMsgUDP(buff, buff1)
					if err != nil {
						log.Println(err)
						return
					}
					log.Println("zhongjian read", target)

					n, _, err = listen.WriteMsgUDP(buff[:length], nil, addr)
					if err != nil {
						log.Println(err)
						return
					}
					log.Println("zhongjian write", local)
				}
			}

			<-connection.waitWrite

			_, _, _ = connection.udp.WriteMsgUDP(buf[:n], nil, nil)
			log.Println("next write data to", target)
		}()
	}

}

func ForwardConnectServer() {
	vnc := &net.UDPAddr{IP: net.ParseIP("192.168.3.215"), Port: 5900}
	source := &net.UDPAddr{IP: net.IPv4zero, Port: 9982}
	server := &net.UDPAddr{IP: net.ParseIP("192.168.3.123"), Port: 51006}
	buf := make([]byte, 4096)
	buf1 := make([]byte, 4096)

	conn, err := net.DialUDP("udp", source, server)
	if err != nil {
		log.Println(err)
		return
	}
	_, _ = conn.Write([]byte("forward"))

	n, _, _ := conn.ReadFromUDP(buf)
	target, _ := net.ResolveUDPAddr("udp", string(buf[:n]))
	log.Println("receive data:", target)
	_ = conn.Close()

	conn, err = net.DialUDP("udp", source, target)
	if err != nil {
		log.Println(target, err)
		return
	}

	time.Sleep(time.Second * 1)
	log.Println("wait one second to send.")

	_, err = conn.Write([]byte("handshake packet."))
	if err != nil {
		log.Println("send handshake err:", err)
		return
	}
	n, _, err = conn.ReadFromUDP(buf)
	if err != nil {
		log.Println("read data err:", err)
		return
	}

	log.Println("get handshake packet yes.", string(buf[:n]))
	lock := new(sync.RWMutex)
	connections := make(map[string]*connectionUDP)

	for {
		n, _, _, addr, err := conn.ReadMsgUDP(buf, buf1)
		if err != nil {
			log.Println("read from", target, "data err:", err)
			return
		}
		log.Println("receive data from", target)

		go func() {
			lock.Lock()
			connection, found := connections[addr.String()]
			if !found {
				connection := new(connectionUDP)
				connection.name = addr.String()
				connection.waitWrite = make(chan bool)
				connection.udp = nil
				connections[addr.String()] = connection
			}
			lock.Unlock()

			if !found {
				udpConn, err := net.DialUDP("udp", nil, vnc)
				if err != nil {
					log.Println("[-] udp-forward: failed to dial:", err)
					return
				}
				log.Println("[*] connect to", vnc, "success.")
				lock.Lock()
				connections[addr.String()].udp = udpConn
				close(connections[addr.String()].waitWrite)
				lock.Unlock()

				_, _, _ = udpConn.WriteMsgUDP(buf[:n], nil, nil)
				log.Println("first send data to vnc")

				for {
					buffer := make([]byte, 4096)
					buffer1 := make([]byte, 4096)
					length, _, _, _, _ := udpConn.ReadMsgUDP(buffer, buffer1)
					log.Println("zhongjian read", vnc)
					// _, _, _ = conn.WriteMsgUDP(buffer[:length], nil, addr)
					_, err = conn.Write(buffer[:length])
					if err != nil {
						log.Println(err)
						return
					}
					log.Println("zhongjian write", addr)
				}
			}

			<-connection.waitWrite
			_, _, _ = connection.udp.WriteMsgUDP(buf[:n], nil, nil)
			log.Println("next write data to", vnc)
		}()
	}
}
