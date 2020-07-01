package forward

import (
	"NetworkGadget/src/main/client"
	"NetworkGadget/src/main/model"
	"crypto/tls"
	"io"
	"log"
	"net"
	"sync"
)

var config *tls.Config = nil

func ListenPortToForwardConnect(base *model.ConfigBase, index string) {
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
			client.OtherWayForward(netForward, accept)
		}()

		go func() {
			defer wg.Done()
			client.OtherWayForward(accept, netForward)
		}()

	} else {
		go normalForward(netForward, accept, wg)
		go normalForward(accept, netForward, wg)
	}

	wg.Wait()
}

func normalForward(src, dst net.Conn, wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("[x] 流量转发错误:%s\n", err.Error())
	}
	//log.Printf("%s -> %s  %d/bytes", dst.RemoteAddr(), src.RemoteAddr(), written)
}
