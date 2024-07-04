package main

import (
	"distribute-system/p2p"
	"fmt"
	"time"
)

func OnPeer(peer p2p.Peer) error {
	fmt.Printf("New peer connected \n")
	return nil
}

func main() {

	tcpOpts := p2p.TCPTransportOps{
		ListenAddr:    ":3001",
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	fileServerOpts := FileServerOpts{
		StorageRoot:       "3000_network",
		PathTransformFunc: CASPathTransformFun,
		Transport:         tr,
	}

	fs := NewFileServer(fileServerOpts)

	go func() {
		time.Sleep(3 * time.Second)
		fs.Stop()
	}()
	if err := fs.Start(); err != nil {
		fmt.Println(err)
	}

	//select {}
}
