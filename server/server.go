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
	var peers []io.Writer
	for _, peer := range s.peers {
		log.Printf("%v broadcasting message to: %v", msg.From, peer.RemoteAddr().String())
		peers = append(peers, peer)
	}
	mw := io.MultiWriter(peers...)

	log.Printf("Payload: %v", msg.Payload)

	//gob.Register(DataMessage{})
	log.Printf("broadcast %v's peer count: %v", msg.From, len(s.peers))

	return gob.NewEncoder(mw).Encode(msg)
}

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

	dataMsg := RPCType.DataMessage{
		Key:  key,
		Data: buf.Bytes(),
	}

	log.Printf("%v broadcasting store key: %v data: %v", s.ListenAddr, dataMsg.Key, string(dataMsg.Data))

	return s.broadcast(&RPCType.Message{
		From:    s.ListenAddr,
		Payload: dataMsg,
	})
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
		case msg := <-s.Transport.Consume():
			//var m RPCType.Message
			//var buf bytes.Buffer
			//if err := gob.NewDecoder(bytes.NewReader(msg.Payload)).Decode(&m); err != nil {
			//	log.Printf("Failed to decode message: %v", err)
			//}
			log.Printf("%v's server received message: %v\n", s.ListenAddr, string(msg.Payload.Data))
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
	//case *DataMessage:
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
