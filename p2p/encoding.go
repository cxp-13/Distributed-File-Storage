package p2p

import (
	"distribute-system/RPCType"
	"encoding/gob"
	"io"
)

type Decoder interface {
	Decode(io.Reader, *RPCType.Message) error
}

type GOBDecoder struct {
}

func (dec GOBDecoder) Decode(r io.Reader, msg *RPCType.Message) error {
	return gob.NewDecoder(r).Decode(msg)
}

//RPCType DefaultDecoder struct {
//}
//
//func (dec DefaultDecoder) Decode(r io.Reader, msg *server.Message) error {
//	_, err := r.Read(msg.Payload)
//	if err != nil {
//		return err
//	}
//	return nil
//}
