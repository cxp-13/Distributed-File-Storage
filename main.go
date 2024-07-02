package main

import (
	"distribute-system/p2p"
	"fmt"
	"log"
)

func main() {

	tcpOpts := p2p.TCPTransportOps{
		ListenAddr:    ":3001",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.NOPDecoder{},
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	go func() {
		for {
			msg := <-tr.Consume()
			fmt.Printf("Received message from: %v\n", msg.From)
			fmt.Printf("Received message: %v\n", string(msg.Payload))

		}
	}()
	if err := tr.ListenAndAccept(); err != nil {
		log.Fatalf("failed to listen and accept: %v", err)
	}

	select {}
}
