package udptypes

type UDPMessage struct {
	Id        int32
	Type      int8
	Length    int16
	Body      []byte
	Signature string
}

type HelloBody struct {
	Extensions int32
	Name       string
}

type DatumBody struct {
	Hash  string
	Value []byte
}
