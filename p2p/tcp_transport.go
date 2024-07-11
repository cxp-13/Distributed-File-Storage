package p2p

import (
	"distribute-system/models"
	"log"
	"net"
)

func (t *TCPTransport) Addr() string {
	return t.ListenAddr
}

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
		if err = t.OnPeer(*peer); err != nil {
			return
		}
	}

	for {
		var rpc models.RPC
		rpc.From = conn.RemoteAddr().String()
		if err := t.Decoder.Decode(conn, &rpc); err != nil {
			log.Printf("decode error: %v", err)
			continue
		}
		log.Printf("transport:%v | handleConn local: %v, remote: %v", t.ListenAddr, conn.LocalAddr().String(), conn.RemoteAddr().String())

		if rpc.Stream {
			log.Printf("transport:%v | %v receive stream signal, stop listen !!!.  waitGroup +1", t.ListenAddr, conn.LocalAddr().String())
			peer.wg.Add(1)
			peer.wg.Wait()
			continue
		}
		t.rpcs <- rpc
	}
}

func init() {

}
