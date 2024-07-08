package p2p

import (
	"distribute-system/RPCType"
	"encoding/gob"
	"io"
	"log"
)

type Decoder interface {
	Decode(io.Reader, *RPCType.RPC) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPCType.RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPCType.RPC) error {
	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	if err != nil {
		log.Fatalf("Decode error : %v", err.Error())
	}
	log.Printf("decode success data size: %v", n)
	msg.Payload = buf[:n]
	return nil
}
