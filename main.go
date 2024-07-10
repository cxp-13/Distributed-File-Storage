package main

import (
	"bytes"
	"distribute-system/p2p"
	"distribute-system/server"
	"fmt"
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
	s2 := makeServer(":4000", ":3000")

	go func() {
		err := s1.Start()
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		err2 := s2.Start()
		if err2 != nil {
			panic(err2)
		}
	}()
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 3; i++ {
		data := bytes.NewReader([]byte("H"))
		err := s2.StoreData(fmt.Sprint("myprivate_", i), data)
		if err != nil {
			panic(err)
		}
	}

	//r, err := s2.Get("myprivate3")
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
