package udptypes

import (
	"errors"
	"net"
)

func NewUDPSocket() (*UDPSock, error) {

	ret, err := net.ListenUDP("udp", &net.UDPAddr{})

	sock := UDPSock{
		Socket: ret,
	}

	return &sock, err
}

func (sock *UDPSock) SendPacket(addr *net.UDPAddr, pack UDPMessage) error {

	bytes := pack.MessageToBytes()

	bytesWritten, err := sock.Socket.WriteTo(bytes, addr)
	if bytesWritten != len(bytes) {
		err = errors.New("message truncated")
	}

	return err
}

func (sock *UDPSock) ReceivePacket() (UDPMessage, net.Addr, error) {

	// (id + type + length) + body max size + signature + 1
	size := 7 + (1 << 16) + 64 + 1

	received := make(UDPMessageBytes, size)

	sizeReceived, from, err := sock.Socket.ReadFrom(received)
	if err != nil {
		return UDPMessage{}, nil, nil
	}
	if err == nil && sizeReceived == size {
		err = errors.New("message truncated")
	}

	msg := received.BytesToMessage()

	return msg, from, err
}
