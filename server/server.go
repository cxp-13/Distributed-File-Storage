package server

import (
	"bytes"
	"distribute-system/RPCType"
	"distribute-system/p2p"
	"encoding/gob"
	"io"
	"log"
	"sync"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts
	store  *Store
	quitch chan struct{}

	peerLock sync.Mutex
	peers    map[string]p2p.Peer
}

func NewFileServer(opts FileServerOpts) *FileServer {
	storeOpts := StoreOpts{
		Root:              opts.StorageRoot,
		PathTransformFunc: opts.PathTransformFunc,
	}

	return &FileServer{
		FileServerOpts: opts,
		store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.Peer),
	}
}

func (s *FileServer) broadcast(msg *RPCType.Message) error {
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(msg)
	if err != nil {
		log.Printf("Failed to encode message: %v", err)
		return err
	}

	data := buf.Bytes()

	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	for addr, peer := range s.peers {
		log.Printf("%v broadcasting message to: %v", s.ListenAddr, addr)
		_, err := peer.Write(data)
		if err != nil {
			log.Printf("Failed to send message to %v: %v", addr, err)
		}
	}
	log.Printf("broadcast %v's peer count: %v", s.ListenAddr, len(s.peers))

	return nil
}

//func (s *FileServer) broadcast(msg *RPCType.Message) error {
//	var peers []io.Writer
//	for _, peer := range s.peers {
//		log.Printf("%v broadcasting message to: %v", s.ListenAddr, peer.RemoteAddr().String())
//		peers = append(peers, peer)
//	}
//	log.Printf("broadcast %v's peer count: %v", s.ListenAddr, len(s.peers))
//
//	mw := io.MultiWriter(peers...)
//
//	return gob.NewEncoder(mw).Encode(msg)
//}

func (s *FileServer) StoreData(key string, data []byte) error {

	if err := s.store.Write(key, bytes.NewReader(data)); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, bytes.NewReader(data))
	if err != nil {
		log.Printf("Failed to read data: %v", err)
		return err
	}

	msg := RPCType.Message{
		Key:  key,
		Data: buf.Bytes(),
	}

	log.Printf("%v broadcasting store key: %v data: %v", s.ListenAddr, msg.Key, string(msg.Data))

	if err = s.broadcast(&msg); err != nil {
		log.Fatalf("%v broadcast fail %v", s.ListenAddr, err.Error())
	}
	//largeData := []byte("this large data")
	//err = s.broadcast(&RPCType.Message{
	//	Key:  key,
	//	Data: largeData,
	//})
	//if err != nil {
	//	log.Fatalf("%v broadcast fail %v", s.ListenAddr, err.Error())
	//}
	return nil
}

func (s *FileServer) Stop() {
	log.Printf("Stopping server...")
	close(s.quitch)
}

func (s *FileServer) OnPeer(p p2p.Peer) error {
	log.Printf("%s add new peer connected: %s", s.ListenAddr, p.RemoteAddr().String())

	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.RemoteAddr().String()] = p

	log.Printf("%s peer count: %v", s.ListenAddr, len(s.peers))

	return nil
}

func (s *FileServer) loop() {
	defer func() {
		log.Println("Server stopped")
		s.Transport.Close()
	}()

	gob.Register(RPCType.Message{})

	for {
		select {
		case rpc := <-s.Transport.Consume():
			log.Printf("%v receive a rpc from %v", s.ListenAddr, rpc.From)
			var msg RPCType.Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("Failed to decode message: %v", err)
			}
			log.Printf("%v's server received message: key:%v data:%v \n", s.ListenAddr, msg.Key, string(msg.Data))
			//if err := s.handleMessage(&m); err != nil {
			//	log.Printf("Failed to handle message: %v", err)
			//}
		case <-s.quitch:

			return
		}

	}
}

func (s *FileServer) handleMessage(msg *RPCType.Message) error {
	//switch v := msg.Payload.(RPCType) {
	//case *Message:
	//	log.Printf("Received data message: %v", v)
	//	//return s.store.Write(v.Key, bytes.NewReader(v.Data))
	//}

	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	// Bootstrap the network
	for _, bootstrapNode := range s.BootstrapNodes {
		log.Printf("%v bootst node %s", s.ListenAddr, bootstrapNode)
		go func(addr string) {
			conn, err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("Failed to dial bootstrap node %s: %v", addr, err)
			}
			err = s.OnPeer(p2p.NewTCPPeer(conn, true))
			if err != nil {
				log.Printf("Failed to handle peer: %v", err)
			}
		}(bootstrapNode)
	}
	return nil
}

func (s *FileServer) Start() error {
	log.Println("Starting server: port:", s.FileServerOpts.ListenAddr)

	if err := s.Transport.ListenAndAccept(s.OnPeer); err != nil {
		log.Fatalf("failed to listen and accept: %v", err)
	}

	log.Printf("%v's bootstrapNodes count: %v", s.ListenAddr, len(s.BootstrapNodes))

	if len(s.BootstrapNodes) > 0 {
		err := s.bootstrapNetwork()
		return err
	}
	s.loop()
	return nil
}
