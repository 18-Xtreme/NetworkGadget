package model

type ConfigBase struct {
	UseTLS           bool
	SrcAddr, DstAddr string
	SrcPort, DstPort int
}