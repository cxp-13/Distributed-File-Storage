package p2p

import (
	"fmt"
	"net"
	"sync"
)

type TCPPeer struct {
	conn     net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		conn:     conn,
		outbound: outbound,
	}
}

func (p TCPPeer) Close() error {
	return p.conn.Close()
}

type TCPTransportOps struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
}

type TCPTransport struct {
	TCPTransportOps
	listener net.Listener
	messages chan RPC
	mu       sync.RWMutex
	peers    map[net.Addr]Peer
}

func NewTCPTransport(opts TCPTransportOps) *TCPTransport {
	return &TCPTransport{
		TCPTransportOps: opts,
		messages:        make(chan RPC),
		peers:           make(map[net.Addr]Peer),
	}
}

func (t *TCPTransport) ListenAndAccept() error {

	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.acceptLoop()

	return nil
}

func (t *TCPTransport) Consume() <-chan RPC {
	return t.messages
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
		}
		fmt.Printf("Accepted connection from %v\n", conn)

		go t.handleConn(conn)
	}
}

type Temp struct {
}

func (t *TCPTransport) handleConn(conn net.Conn) {
	tcpPeer := NewTCPPeer(conn, true)
	fmt.Printf("Handling connection from %v\n", conn.RemoteAddr())
	t.peers[conn.RemoteAddr()] = tcpPeer

	if err := t.HandshakeFunc(conn); err != nil {
		fmt.Printf("Error during handshake: %v\n", err)
		conn.Close()
		return
	}

	msg := &RPC{
		From:    conn.RemoteAddr().String(),
		Payload: make([]byte, 1024),
	}

	for {
		if err := t.Decoder.Decode(conn, msg); err != nil {
			fmt.Printf("Error during decoding: %v\n", err)
			continue
		}
		//fmt.Printf("Received message from: %v\n", msg.From)
		//fmt.Printf("Received message: %v\n", string(msg.Payload))
		t.messages <- *msg
	}

}
