package server

import (
	"bytes"
	"distribute-system/models"
	"distribute-system/p2p"
	"encoding/gob"
	"errors"
	"fmt"
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

func (s *FileServer) broadcast(msg *models.Message) error {
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
		if err = peer.Send(data); err != nil {
			log.Fatalf("%v send message to %v fail: %v", s.ListenAddr, addr, err.Error())
		}
		log.Printf("%v send message to %v", s.ListenAddr, addr)
		//peer.CloseStream()
	}
	return nil
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	fileBuf := new(bytes.Buffer)
	er := io.TeeReader(r, fileBuf)

	size, err := s.store.Write(key, er)
	if err != nil {
		return err
	}

	msg := models.Message{
		Payload: models.StoreFileMessage{
			Key:  key,
			Size: size,
		},
	}

	if err = s.broadcast(&msg); err != nil {
		log.Fatalf("%v broadcast fail %v", s.ListenAddr, err.Error())
	}

	//time.Sleep(3 * time.Second)

	if err = s.sendDataToPeers(fileBuf); err != nil {
		log.Fatalf("%v send data to peers fail %v", s.ListenAddr, err.Error())
	}

	return nil
}

func (s *FileServer) sendDataToPeers(fileBuf *bytes.Buffer) error {
	var data = fileBuf.Bytes()
	for addr, peer := range s.peers {
		if err := peer.Send(data); err != nil {
			return errors.New(fmt.Sprintf("%v send data to %v fail: %v", s.ListenAddr, addr, err.Error()))
		}
		log.Printf("%v send data:%v to %v", s.ListenAddr, string(data), addr)
	}
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

	for {
		select {
		case rpc := <-s.Transport.Consume():
			log.Printf("%v receive a rpc from %v", s.ListenAddr, rpc.From)
			var msg models.Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("Failed to decode message: %v", err)
			}
			if err := s.handleMessage(rpc.From, msg); err != nil {
				log.Printf("Failed to handle message: %v", err)
			}
		case <-s.quitch:
			return
		}

	}
}

func (s *FileServer) handleMessage(from string, msg models.Message) error {
	switch v := msg.Payload.(type) {
	case models.StoreFileMessage:
		err := s.handleStoreFileMessage(from, v)
		return err
	default:
		return errors.New("Unknown message type")
	}
}

func (s *FileServer) handleStoreFileMessage(from string, msg models.StoreFileMessage) error {
	peer := s.peers[from]
	if peer == nil {
		return errors.New(from + "peer not found")
	}
	_, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	log.Printf("%v store file %v success", s.ListenAddr, msg.Key)
	peer.CloseStream()

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

	if err := s.Transport.ListenAndAccept(); err != nil {
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

func init() {
	gob.Register(models.Message{})
	gob.Register(models.StoreFileMessage{})

}
