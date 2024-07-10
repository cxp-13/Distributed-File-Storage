package models

const (
	IncomingMessage = 0x1
	IncomingStream  = 0x2
)

type RPC struct {
	From    string
	Payload []byte
	Stream  bool
}

type Message struct {
	Payload any
}

type StoreFileMessage struct {
	Key  string
	Size int64
}

type GetFileMessage struct {
	Key string
}
