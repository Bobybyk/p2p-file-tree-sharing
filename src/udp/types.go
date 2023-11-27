package udptypes

import (
	"net"
	"protocoles-internet-2023/filestructure"
	"sync"
	"time"
)

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
	NoDatum                   = 133
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
	Lock         sync.Mutex
	PacketSender chan SchedulerEntry
	PeerDatabase map[string]PeerInfo
}

type PeerInfo struct {
	Name          string
	PublicKey     []byte
	Root          [32]byte
	TreeStructure filestructure.Directory
}
