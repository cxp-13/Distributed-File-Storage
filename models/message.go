package models

type RPC struct {
	From    string
	Payload []byte
}

type Message struct {
	Payload any
}

type StoreFileMessage struct {
	Key  string
	Size int64
}
