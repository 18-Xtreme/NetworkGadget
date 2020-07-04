package model

// 协议头标识
var ProtocolHeader = "size"

type ConfigBase struct {
	UseTLS           bool
	SrcAddr, DstAddr string
	SrcPort, DstPort int
}

type NextProxyNode struct {
	NodeAddr string
	NodePort int
}
