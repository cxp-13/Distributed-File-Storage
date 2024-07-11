package p2p

import (
	"bytes"
	"distribute-system/models"
	"io"
	"log"
	"net"
	"sync"
)

type Peer interface {
	net.Conn
	Send([]byte) error
	ReadStream() (io.Reader, error)
	CloseStream()
}

type TCPPeer struct {
	net.Conn
	Outbound bool
	wg       *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		Outbound: outbound,
		wg:       &sync.WaitGroup{},
	}
}

func (p *TCPPeer) Send(data []byte) error {
	_, err := p.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (p *TCPPeer) ReadStream() (io.Reader, error) {
	buffer := make([]byte, 1024)
	n, err := p.Read(buffer)
	if err != nil {
		log.Printf("Error reading data from peer %s, %v", p.LocalAddr().String(), err)
		return nil, err
	}
	return bytes.NewReader(buffer[:n]), nil
}

func (p *TCPPeer) CloseStream() {
	p.wg.Done()
}

type TCPTransportOps struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	OnPeer        func(TCPPeer) error
}

type TCPTransport struct {
	TCPTransportOps
	listener net.Listener
	rpcs     chan models.RPC
	mu       sync.RWMutex
}

func NewTCPTransport(opts TCPTransportOps) *TCPTransport {
	return &TCPTransport{
		TCPTransportOps: opts,
		rpcs:            make(chan models.RPC),
	}
}
