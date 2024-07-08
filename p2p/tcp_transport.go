package p2p

import (
	"distribute-system/RPCType"
	"log"
	"net"
	"sync"
)

type TCPPeer struct {
	net.Conn
	outbound bool
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
	}
}

func (p TCPPeer) Send(b []byte) error {
	n, err := p.Write(b)
	if err != nil || n != len(b) {

		log.Fatalf("failed to send message: %v", err)
	}
	return nil
}

//func (p TCPPeer) RemoteAddr() net.Addr {
//	return p.conn.RemoteAddr()
//}
//
//func (p TCPPeer) Close() error {
//	return p.conn.Close()
//}

type TCPTransportOps struct {
	ListenAddr    string
	HandshakeFunc HandshakeFunc
	Decoder       Decoder
	//OnPeer        func(Peer) error
}

type TCPTransport struct {
	TCPTransportOps
	listener net.Listener
	messages chan RPCType.RPC
	mu       sync.RWMutex
}

func NewTCPTransport(opts TCPTransportOps) *TCPTransport {
	return &TCPTransport{
		TCPTransportOps: opts,
		messages:        make(chan RPCType.RPC),
	}
}

//func (t *TCPTransport) ListenAddr() string {
//	return t.ListenAddr
//}

func (t *TCPTransport) Dial(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	go t.handleConn(conn)
	return conn, nil
}

func (t *TCPTransport) ListenAndAccept(f func(p Peer) error) error {

	var err error

	t.listener, err = net.Listen("tcp", t.ListenAddr)
	if err != nil {
		return err
	}

	go t.acceptLoop(f)

	return nil
}

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Consume() <-chan RPCType.RPC {
	return t.messages
}

func (t *TCPTransport) acceptLoop(f func(p Peer) error) {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
		}
		log.Printf("%v accepted connection from %v\n", t.ListenAddr, conn.RemoteAddr())
		err = f(NewTCPPeer(conn, false))
		if err != nil {
			log.Printf("Error during OnPeer: %v\n", err)
		}
		go t.handleConn(conn)
	}
}

type Temp struct {
}

func (t *TCPTransport) handleConn(conn net.Conn) {

	var err error

	defer func() {
		log.Printf("Closing connection from %v\n", conn.RemoteAddr())
		conn.Close()
	}()

	//tcpPeer := NewTCPPeer(conn, outbound)

	//if t.OnPeer != nil {
	//	if err = t.OnPeer(tcpPeer); err != nil {
	//		log.Printf("Error during OnPeer: %v\n", err)
	//		return
	//	}
	//}

	if err = t.HandshakeFunc(conn); err != nil {
		log.Printf("Error during handshake: %v\n", err)
		conn.Close()
		return
	}

	for {
		var rpc RPCType.RPC
		rpc.From = conn.RemoteAddr().String()
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			log.Printf("Error during decoding: %v\n", err)
		}
		log.Printf("%v's peer %v received message: %v\n", t.ListenAddr, conn.RemoteAddr())
		t.messages <- rpc
	}

}
