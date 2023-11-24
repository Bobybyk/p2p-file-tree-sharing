package udptypes

const (
	NoOp                uint8 = 0
	Error                     = 1
	Hello                     = 2
	PublicKey                 = 3
	Root                      = 4
	GetDatum                  = 5
	NatTraversalRequest       = 6
	NatTraversal              = 7
	ErrorReply                = 128
	HelloReply                = 129
	PublicKeyReply            = 130
	RootReply                 = 131
	Datum                     = 132
)

type UDPMessageBytes []byte

type UDPMessage struct {
	Id        uint32
	Type      uint8
	Length    uint16
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
