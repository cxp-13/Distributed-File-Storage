package main

import (
	"distribute-system/p2p"
	"fmt"
	"log"
)

type FileServerOpts struct {
	ListenAddr        string
	StorageRoot       string
	PathTransformFunc PathTransformFunc
	Transport         p2p.Transport
}

type FileServer struct {
	FileServerOpts
	store  *Store
	quitch chan struct{}
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
	}
}

func (s FileServer) Stop() {
	log.Printf("Stopping server...")
	close(s.quitch)
}

func (s FileServer) loop() {
	defer func() {
		log.Println("Server stopped")
		s.Transport.Close()
	}()

	for {
		select {
		case msg := <-s.Transport.Consume():
			fmt.Printf("Received message from: %v\n", msg.From)
			fmt.Printf("Received message: %v\n", string(msg.Payload))
		case <-s.quitch:

			return
		}

	}
}

func (s *FileServer) Start() error {

	if err := s.Transport.ListenAndAccept(); err != nil {
		log.Fatalf("failed to listen and accept: %v", err)
	}

	s.loop()
	return nil
}
