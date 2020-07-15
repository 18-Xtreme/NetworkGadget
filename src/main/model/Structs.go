package model

import "net"

// 协议头标识
var ProtocolHeader = "size"

type ConfigBase struct {
	IsUdp            bool
	UseTLS           bool
	SrcAddr, DstAddr string
	SrcPort, DstPort int
}

type NextProxyNode struct {
	NodeAddr string
	NodePort int
}

type ConnectionUdp struct {
	Name     string
	Udp      *net.UDPConn
	WaitChan chan bool
}
