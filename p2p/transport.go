package p2p

import (
	"distribute-system/models"
	"net"
)

type Peer interface {
	net.Conn
	Send([]byte) error
	//RemoteAddr() net.Addr
	CloseStream()
}

type Transport interface {
	Dial(string) (net.Conn, error)
	ListenAndAccept() error
	Consume() <-chan models.RPC
	Close() error
	//ListenAddr() string
}
