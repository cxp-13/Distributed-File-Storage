package server

import (
	"bytes"
	"distribute-system/crypto"
	"distribute-system/models"
	"distribute-system/p2p"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
	"time"
)

type FileServerOpts struct {
	EncKey            []byte
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
	BootstrapNodes    []string
}

type FileServer struct {
	FileServerOpts
	Store  *Store
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
		Store:          NewStore(storeOpts),
		quitch:         make(chan struct{}),
		peers:          make(map[string]p2p.TCPPeer),
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
		if peer.Outbound {
			peer.Send([]byte{models.IncomingMessage})
			if err = peer.Send(data); err != nil {
				log.Fatalf("%v send message to %v fail: %v", s.ListenAddr, addr, err.Error())
			}
			log.Printf("%v send message to %v", s.ListenAddr, addr)
		}
	}
	return nil
}

func (s *FileServer) Get(key string) (io.Reader, error) {
	if s.Store.Has(key) {
		return s.Store.Read(key)
	}

	log.Printf("%v has no key %v, fetch from network", s.ListenAddr, key)

	msg := models.Message{
		Payload: models.GetFileMessage{
			Key: key,
		},
	}

	if err := s.broadcast(&msg); err != nil {
		log.Fatalf("%v broadcast get msg fail %v", s.ListenAddr, err.Error())
		return nil, err
	}
	time.Sleep(time.Second * 1)

	//var multiReader io.Reader
	for addr, peer := range s.peers {
		if peer.Outbound {
			var fileSize int64
			if err := binary.Read(peer, binary.LittleEndian, &fileSize); err != nil {
				log.Printf("%v read file size from %v fail: %v", s.ListenAddr, addr, err.Error())
				continue
			}
			//fileReader, err := peer.ReadStream()
			//if err != nil {
			//	return nil, errors.New(fmt.Sprintf("%v fetch data from %v fail: %v", s.ListenAddr, addr, err.Error()))
			//}
			_, err := s.Store.WriteDecrypt(s.EncKey, key, io.LimitReader(peer, fileSize))
			//_, err = s.Store.Write(key, fileReader)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("%v Store data from %v fail: %v", s.ListenAddr, addr, err.Error()))
			}
			log.Printf("server: %v|closing %v stream,  -1", s.ListenAddr, addr)
			peer.CloseStream()
		}
	}
	return s.Store.Read(key)
}

func (s *FileServer) StoreData(key string, r io.Reader) error {
	fileBuf := new(bytes.Buffer)
	er := io.TeeReader(r, fileBuf)

	size, err := s.Store.Write(key, er)
	if err != nil {
		return err
	}

	msg := models.Message{
		Payload: models.StoreFileMessage{
			Key:  key,
			Size: size + 16,
		},
	}

	if err = s.broadcast(&msg); err != nil {
		log.Fatalf("%v broadcast Store file fail %v", s.ListenAddr, err.Error())
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
		if _, err := crypto.CopyEncrypt(s.EncKey, fileBuf, peer); err != nil {
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
		log.Printf("%v receive a Store file message from %v", s.ListenAddr, from)
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
	peer, has := s.peers[from]
	if !has {
		return errors.New(from + "peer not found")
	}
	if !s.Store.Has(msg.Key) {
		return errors.New(peer.LocalAddr().String() + "has no key " + msg.Key)
	}
	file, err := s.Store.Read(msg.Key)
	if err != nil {
		return err
	}
	if err = peer.Send([]byte{models.IncomingStream}); err != nil {
		return err
	}
	stat, _ := file.Stat()
	if err := binary.Write(peer, binary.LittleEndian, stat.Size()); err != nil {
		return err
	}
	_, err = io.Copy(peer, file)
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
	_, err := s.Store.Write(msg.Key, io.LimitReader(peer, msg.Size))
	if err != nil {
		return err
	}
	log.Printf("%v server | %v Store file %v success, close stream, waitGroup -1", s.ListenAddr, peer.LocalAddr().String(), msg.Key)
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
