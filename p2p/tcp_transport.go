package p2p

import (
	"distribute-system/models"
	"log"
	"net"
	"sync"
)

type TCPPeer struct {
	net.Conn
	outbound bool
	wg       *sync.WaitGroup
}

func NewTCPPeer(conn net.Conn, outbound bool) *TCPPeer {
	return &TCPPeer{
		Conn:     conn,
		outbound: outbound,
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

func (p *TCPPeer) CloseStream() {
	log.Printf("Closing stream for peer %s,  -1", p.LocalAddr().String())
	p.wg.Done()
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
	OnPeer        func(Peer) error
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

//func (t *TCPTransport) ListenAddr() string {
//	return t.ListenAddr
//}

func (t *TCPTransport) Dial(addr string) (net.Conn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	go t.handleConn(conn, true)
	return conn, nil
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

func (t *TCPTransport) Close() error {
	return t.listener.Close()
}

func (t *TCPTransport) Consume() <-chan models.RPC {
	return t.rpcs
}

func (t *TCPTransport) acceptLoop() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v\n", err)
		}
		log.Printf("%v accepted connection from %v\n", t.ListenAddr, conn.RemoteAddr())
		go t.handleConn(conn, false)
	}
}

func (t *TCPTransport) handleConn(conn net.Conn, outbound bool) {
	var err error
	peer := NewTCPPeer(conn, outbound)

	defer func() {
		log.Printf("Closing connection from %v\n", conn.RemoteAddr())
		conn.Close()
	}()

	if err = t.HandshakeFunc(conn); err != nil {
		log.Fatalf("Error during handshake: %v\n", err)
	}

	if t.OnPeer != nil {
		if err = t.OnPeer(peer); err != nil {
			return
		}
	}

	for {
		var rpc models.RPC
		rpc.From = conn.RemoteAddr().String()
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			log.Printf("Error during decoding: %v\n", err)
		}
		peer.wg.Add(1)
		log.Printf("%v wg +1", conn.LocalAddr().String())
		log.Printf("%v received peer %v", t.ListenAddr, conn.LocalAddr().String())

		t.rpcs <- rpc
		peer.wg.Wait()

	}

}

func init() {
}
