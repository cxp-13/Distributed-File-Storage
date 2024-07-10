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
	"time"
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
	peers    map[string]p2p.TCPPeer
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
		peers:          make(map[string]p2p.TCPPeer),
	}
}

func (s *FileServer) broadcast(msg *models.Message, msgType byte) error {
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
		if peer.Outbound {
			err := peer.Send([]byte{msgType})
			if err != nil {
				return err
			}
			if err = peer.Send(data); err != nil {
				log.Fatalf("%v send message to %v fail: %v", s.ListenAddr, addr, err.Error())
			}
			log.Printf("%v send message to %v", s.ListenAddr, addr)
			//peer.CloseStream()
		}
	}
	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.store.Has(key) {
		return s.store.Read(key)
	}

	log.Printf("%v has no key %v, fetch from network", s.ListenAddr, key)

	msg := models.Message{
		Payload: models.GetFileMessage{
			Key: key,
		},
	}

	if err := s.broadcast(&msg, models.IncomingMessage); err != nil {
		log.Fatalf("%v broadcast get msg fail %v", s.ListenAddr, err.Error())
		return nil, err
	}
	time.Sleep(time.Second * 2)

	//var multiReader io.Reader
	for addr, peer := range s.peers {
		if peer.Outbound {
			data, err := peer.FetchData()
			if err != nil {
				return nil, errors.New(fmt.Sprintf("%v fetch data from %v fail: %v", s.ListenAddr, addr, err.Error()))
			}
			if len(data) == 0 {
				log.Printf("%v fetch data from %v fail: data is empty", s.ListenAddr, addr)
				continue
			}
			reader := bytes.NewReader(data)
			//multiReader = io.MultiReader(multiReader, reader)
			time.Sleep(time.Second * 4)
			log.Printf("server: %v|closing %v stream,  -1", s.ListenAddr, addr)
			peer.CloseStream()
			return reader, nil
		}
	}
	return nil, nil
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

	if err = s.broadcast(&msg, models.IncomingMessage); err != nil {
		log.Fatalf("%v broadcast store file fail %v", s.ListenAddr, err.Error())
	}

	time.Sleep(time.Millisecond * 500)

	if err = s.sendDataToPeers(fileBuf); err != nil {
		log.Fatalf("%v send data to peers fail %v", s.ListenAddr, err.Error())
	}

	return nil
}

func (s *FileServer) sendDataToPeers(fileBuf *bytes.Buffer) error {
	var data = fileBuf.Bytes()
	for addr, peer := range s.peers {
		if err := peer.Send([]byte{models.IncomingStream}); err != nil {
			return errors.New(fmt.Sprintf("%v send IncomingStream to %v fail: %v", s.ListenAddr, addr, err.Error()))
		}
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

func (s *FileServer) OnPeer(p p2p.TCPPeer) error {
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
			log.Printf("%v server receive a rpc from %v", s.ListenAddr, rpc.From)

			var msg models.Message

			if err := gob.NewDecoder(bytes.NewReader(rpc.Payload)).Decode(&msg); err != nil {
				log.Printf("failed to decode message: %v", err)
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
		log.Printf("%v receive a store file message from %v", s.ListenAddr, from)
		err := s.handleStoreFileMessage(from, v)
		return err

	case models.GetFileMessage:

		log.Printf("%v receive a get file message from %v", s.ListenAddr, from)
		err := s.handleGetFileMessage(from, v)
		return err
	default:
		return errors.New("Unknown message type")
	}
}

func (s *FileServer) handleGetFileMessage(from string, msg models.GetFileMessage) error {
	peer := s.peers[from]
	//if peer == nil {
	//	return errors.New(from + "peer not found")
	//}
	if !s.store.Has(msg.Key) {
		return errors.New(peer.LocalAddr().String() + "has no key " + msg.Key)
	}
	reader, err := s.store.Read(msg.Key)
	if err != nil {
		return err
	}
	if err = peer.Send([]byte{models.IncomingStream}); err != nil {
		return err
	}
	_, err = io.Copy(peer, reader)
	if err != nil {
		return err
	}
	return nil
}

func (s *FileServer) handleStoreFileMessage(from string, msg models.StoreFileMessage) error {
	peer := s.peers[from]
	//if peer == nil {
	//	return errors.New(from + "peer not found")
	//}
	_, err := s.store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	log.Printf("%v server | %v store file %v success, close stream, waitGroup -1", s.ListenAddr, peer.LocalAddr().String(), msg.Key)
	peer.CloseStream()
	return nil
}

func (s *FileServer) bootstrapNetwork() error {
	// Bootstrap the network
	for _, bootstrapNode := range s.BootstrapNodes {
		go func(addr string) {
			conn, err := s.Transport.Dial(addr)
			if err != nil {
				log.Printf("Failed to dial bootstrap node %s: %v", addr, err)
			}
			log.Printf("%v bootst beer local: %s remote: %s", s.ListenAddr, conn.LocalAddr().String(), conn.RemoteAddr().String())
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
	gob.Register(models.GetFileMessage{})

}
