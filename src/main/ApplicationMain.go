package main

import (
	"NetworkGadget/src/main/client"
	"NetworkGadget/src/main/forward"
	"NetworkGadget/src/main/model"
	"NetworkGadget/src/main/proxy"
	"NetworkGadget/src/main/server"
	"NetworkGadget/src/test"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	localAddr  string
	targetAddr string
)

func main() {
	//start()
	test.TestMain3()
}

func start() {
	base := new(model.ConfigBase)
	base.UseTLS = false
	l, t := 2, 3
	if (len(os.Args) == 4 || len(os.Args) == 6) && os.Args[1] == "-forward" {
		var index = "0"
		if os.Args[2] == "--tls" {
			base.UseTLS = true
			index = os.Args[3]
			l, t = 4, 5
		}
		localAddr = os.Args[l]
		targetAddr = os.Args[t]
		fillStructure(base)
		forward.ListenPortToForwardConnect(base, index)
	} else if len(os.Args) == 4 || len(os.Args) == 5 {
		if os.Args[2] == "--tls" {
			base.UseTLS = true
			l, t = 3, 4
		}
		if os.Args[1] == "-listen" {
			targetAddr = os.Args[t]
			localAddr = os.Args[l]
			fillStructure(base)
			server.MainServer(base)
		} else if os.Args[1] == "-connect" {
			targetAddr = os.Args[l]
			localAddr = os.Args[t]
			fillStructure(base)
			client.MainClient(base)
		}
	} else if (len(os.Args) == 3 || len(os.Args) == 4) && os.Args[1] == "-proxy" {
		if os.Args[2] == "--tls" {
			base.UseTLS = true
			l = 3
		}
		localAddr = os.Args[l]
		fillStructure(base)
		proxy.StartProxy(base)
	} else {
		usage()
		os.Exit(1)
	}

}

func usage() {
	fmt.Println("[帮助手册:]")
	fmt.Println("  ng -<forward|listen|connect> <选项>")
	fmt.Println("\n[选项:]")
	fmt.Println("  -forward [--tls (1|2|3)] <监听端口> <IP|转发端口>")
	fmt.Println("  -listen [--tls] <监听端口> <转发端口>")
	fmt.Println("  -connect [--tls] <IP|服务器转发端口> <IP|实际转发端口>")
	fmt.Println("  -proxy [--tls] <IP|代理端口>")
	fmt.Println("  --tls (1|2|3) 使用tls加密 (1:第一个端口加密||2:第二个端口加密||3:全部加密)")
	fmt.Println("\n[例如:]")
	fmt.Println("  ng -forward 1234 3389")
	fmt.Println("  ng -forward 1234 x.x.x.x:3389")
	fmt.Println("  ng -forward --tls 1 1234 3389")
	fmt.Println("  ng -listen 51006 51007")
	fmt.Println("  ng -listen --tls 51006 51007")
	fmt.Println("  ng -connect 51007 3389")
	fmt.Println("  ng -connect x.x.x.x:51007 x.x.x.x:3389")
	fmt.Println("  ng -proxy 51007")
	fmt.Println("  ng -proxy --tls 51007")
}

func fillStructure(base *model.ConfigBase) {
	base.SrcAddr, base.SrcPort = formatParseValue(localAddr)
	base.DstAddr, base.DstPort = formatParseValue(targetAddr)

}

func formatParseValue(argv string) (string, int) {
	if argv == "" {
		return "", 0
	}

	if strings.Contains(argv, ":") {
		v, _ := strconv.Atoi(strings.Split(argv, ":")[1])
		return argv, v
	} else {
		v, _ := strconv.Atoi(argv)
		return fmt.Sprintf(":%s", argv), v
	}

}
