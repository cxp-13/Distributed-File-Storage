package RPCType

type RPC struct {
	From    string
	Payload []byte
}

//type Message struct {
//	From    string
//	Payload Message
//}

type Message struct {
	Key  string
	Data []byte
}
