package server

import "net"

type ForwardModel struct {
	accept        net.Conn
	acceptAddTime int64
	tunnel        net.Conn
}
