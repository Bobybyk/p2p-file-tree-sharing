package udptypes

import (
	"crypto/ecdsa"
	"net"
	"protocoles-internet-2023/filestructure"
	"sync"
	"time"
)

// DO NOT CHANGE ORDER
const (
	//requests
	NoOp      uint8 = 0
	Error           = 1
	Hello           = 2
	PublicKey       = 3
	Root            = 4
	GetDatum        = 5

	//RESPONSES
	NatTraversalRequest = 6
	NatTraversal        = 7
	ErrorReply          = 128
	HelloReply          = 129
	PublicKeyReply      = 130
	RootReply           = 131
	Datum               = 132
	NoDatum             = 133
)

type UDPMessageBytes []byte

type UDPMessage struct {
	Id         uint32
	Type       uint8
	Length     uint16
	Body       []byte
	Signature  []byte
	PrivateKey *ecdsa.PrivateKey //optional, if public key is provided, then the message will be signed
}

type HelloBody struct {
	Extensions int32
	Name       string
}

type DatumBody struct {
	Hash  [32]byte
	Value []byte
}

type UDPSock struct {
	Socket *net.UDPConn
}

type SchedulerEntry struct {
	Time   time.Time
	To     *net.UDPAddr
	From   net.Addr
	Packet UDPMessage
}

type Scheduler struct {
	Lock           sync.Mutex
	Socket         UDPSock
	PacketReceiver chan SchedulerEntry
	PeerDatabase   map[string]*PeerInfo
	PrivateKey     *ecdsa.PrivateKey
	PublicKey      *ecdsa.PublicKey
	ExportedFiles  *filestructure.Directory
}

type PeerInfo struct {
	Name      string
	PublicKey []byte
	Root      [32]byte
}
