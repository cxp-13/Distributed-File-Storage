package p2p

import (
	"distribute-system/RPCType"
	"net"
)

type Peer interface {
	net.Conn
	Send([]byte) error
	//RemoteAddr() net.Addr
	//Close() error
}

type Transport interface {
	Dial(string) (net.Conn, error)
	ListenAndAccept(func(p Peer) error) error
	Consume() <-chan RPCType.RPC
	Close() error
	//ListenAddr() string
}
