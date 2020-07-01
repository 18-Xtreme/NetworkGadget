package main

import (
	"NetworkGadget/src/main/client"
	"NetworkGadget/src/main/forward"
	"NetworkGadget/src/main/model"
	"NetworkGadget/src/main/server"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var (
	targetAddr string
	localAddr  string
)

func main() {
	start()
}

func start() {
	base := new(model.ConfigBase)
	if len(os.Args) > 2 && os.Args[1] == "-forward" {
		localAddr = os.Args[2]
		targetAddr = os.Args[3]
		fillStructure(base)
		forward.ListenPortToForwardConnect(base)
	} else if len(os.Args) > 3 && os.Args[1] == "-listen" {
		targetAddr = os.Args[3]
		localAddr = os.Args[2]
		fillStructure(base)
		server.MainServer(base)
	} else if len(os.Args) > 3 && os.Args[1] == "-connect" {
		targetAddr = os.Args[2]
		localAddr = os.Args[3]
		fillStructure(base)
		client.MainClient(base)
	} else {
		usage()
		os.Exit(1)
	}

}

func usage() {
	fmt.Println("[帮助手册:]")
	fmt.Println("  ns -<forward|listen|connect> <选项>")
	fmt.Println("\n[选项:]")
	fmt.Println("  -forward <本地端口> <转发端口>")
	fmt.Println("  -listen <监听端口> <转发端口>")
	fmt.Println("  -connect <服务器转发端口> <本地转发端口>")
	fmt.Println("\n[例如:]")
	fmt.Println("  ns -forward 1234 3389")
	fmt.Println("  ns -forward 1234 x.x.x.x:3389")
	fmt.Println("  ns -listen 51006 51007")
	fmt.Println("  ns -connect 51007 3389")
	fmt.Println("  ns -connect x.x.x.x:51007 x.x.x.x:3389")
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
