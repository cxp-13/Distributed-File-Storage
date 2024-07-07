package RPCType

type RPC struct {
	From    string
	Payload []byte
}

type Message struct {
	From    string
	Payload DataMessage
}

type DataMessage struct {
	Key  string
	Data []byte
}
