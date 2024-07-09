package p2p

import (
	"distribute-system/models"
	"encoding/gob"
	"io"
	"log"
)

type Decoder interface {
	Decode(io.Reader, *models.RPC) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, msg *models.RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, rpc *models.RPC) error {
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		log.Fatalf("Decode error : %v", err.Error())
	}
	log.Printf("decode success data size: %v", n)
	rpc.Payload = buf[:n]
	return nil
}
