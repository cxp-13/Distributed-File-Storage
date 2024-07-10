package p2p

import (
	"distribute-system/models"
	"encoding/gob"
	"errors"
	"io"
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

	peekBuf := make([]byte, 1)
	if _, err := r.Read(peekBuf); err != nil {
		return errors.New("peek read fail:" + err.Error())
	}

	isStream := peekBuf[0] == models.IncomingStream

	if isStream {
		rpc.Stream = true
	} else {
		rpc.Stream = false
		buf := make([]byte, 1024)
		n, err := r.Read(buf)
		if err != nil {
			//log.Fatalf("decode error : %v", err.Error())
			return err
		}
		rpc.Payload = buf[:n]
	}

	return nil
}
