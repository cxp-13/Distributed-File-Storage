package p2p

import "errors"

var ErrInvalidHandshake = errors.New("invalid handshake")

type HandshakeFunc func(any) error

func NOPHandshakeFunc(any) error { return nil }
