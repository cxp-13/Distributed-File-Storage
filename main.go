package main

import (
	"bytes"
	"distribute-system/crypto"
	"distribute-system/p2p"
	"distribute-system/server"
	"log"
	"strings"
	"time"
)

func makeServer(listenAddr string, nodes ...string) *server.FileServer {
	tcpOpts := p2p.TCPTransportOps{
		ListenAddr:    listenAddr,
		HandshakeFunc: p2p.NOPHandshakeFunc,
		Decoder:       p2p.DefaultDecoder{},
		//OnPeer:        OnPeer,
	}

	tr := p2p.NewTCPTransport(tcpOpts)

	fileServerOpts := server.FileServerOpts{
		EncKey:            crypto.NewEncryptionKey(),
		ListenAddr:        listenAddr,
		StorageRoot:       "store_" + strings.TrimPrefix(listenAddr, ":"),
		PathTransformFunc: server.CASPathTransformFun,
		Transport:         tr,
		BootstrapNodes:    nodes,
	}

	s := server.NewFileServer(fileServerOpts)

	tr.OnPeer = s.OnPeer

	return s
}

func main() {

	s1 := makeServer(":3000")
	s2 := makeServer(":4000")
	s3 := makeServer(":5000", ":3000", ":4000")

	go func() {
		err := s1.Start()
		if err != nil {
			log.Printf("Error starting server 1: %v", err)
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)
	go func() {
		err := s2.Start()
		if err != nil {
			log.Printf("Error starting server 2: %v", err)
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)
	go func() {
		err := s3.Start()
		if err != nil {
			log.Printf("Error starting server 3: %v", err)
			panic(err)
		}
	}()
	time.Sleep(3 * time.Second)

	key := "coolPicure"

	data := bytes.NewReader([]byte("Hello world"))

	if err := s3.StoreData(key, data); err != nil {
		panic(err)
	}

	//if err := s2.Store.Delete(key); err != nil {
	//	panic(err)
	//}
	//
	//r, err := s2.Get(key)
	//if err != nil {
	//	panic(err)
	//}
	//b, err := ioutil.ReadAll(r)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Printf("Got data: %s", string(b))
	select {}

}
