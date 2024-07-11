package p2p

import (
	"distribute-system/models"
	"net"
)

type Transport interface {
	Dial(string) (net.Conn, error)
	ListenAndAccept() error
	Consume() <-chan models.RPC
	Close() error
	Addr() string
	//ListenAddr() string
}
