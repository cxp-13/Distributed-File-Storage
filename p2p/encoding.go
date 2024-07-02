package p2p

import (
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPC) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPC) error {
	return gob.NewDecoder(r).Decode(msg)
}

type DefaultDecoder struct {
}

func (dec DefaultDecoder) Decode(r io.Reader, msg *RPC) error {
	//buf := make([]byte, 1024)

	_, err := r.Read(msg.Payload)

	if err != nil {
		return err
	}

	//fmt.Printf("%v", string(msg.Payload[:n]))

	return nil
}
