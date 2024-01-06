package udptypes

import (
	"fmt"
	"math/rand"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/crypto"
)

func (sched *Scheduler) SendNoOp(dest *net.UDPAddr) {

	if config.Debug {
		fmt.Print("Sending NoOp to ")
		peer, ok := sched.PeerDatabase[dest.String()]
		if ok {
			fmt.Println(peer.Name)
		} else {
			fmt.Println(dest.String())
		}
	}

	msg := UDPMessage{
		Id:     uint32(rand.Int31()),
		Type:   NoOp,
		Length: 0,
	}
	err := sched.Socket.SendPacket(msg, dest)
	if err != nil {
		return
	}
}

func (sched *Scheduler) SendHello(dest *net.UDPAddr) {

	if config.Debug {
		fmt.Print("Sending Hello to ")
		peer, ok := sched.PeerDatabase[dest.String()]
		if ok {
			fmt.Println(peer.Name)
		} else {
			fmt.Println(dest.String())
		}
	}

	body := HelloBody{
		Name:       config.ClientName,
		Extensions: 0,
	}.HelloBodyToBytes()

	msg := UDPMessage{
		Id:         uint32(rand.Int31()),
		Type:       Hello,
		Length:     uint16(len(body)),
		Body:       body,
		PrivateKey: sched.PrivateKey,
	}
	_, err := sched.SendPacket(msg, dest)
	if err != nil {
		fmt.Println("SendHello: ", err.Error())
		return
	}
}

func (sched *Scheduler) SendHelloReply(dest *net.UDPAddr, id uint32) {

	body := HelloBody{
		Name:       config.ClientName,
		Extensions: 0,
	}.HelloBodyToBytes()

	msg := UDPMessage{
		Id:         id,
		Type:       HelloReply,
		Length:     uint16(len(body)),
		Body:       body,
		PrivateKey: sched.PrivateKey,
	}
	err := sched.Socket.SendPacket(msg, dest)
	if err != nil {
		return
	}

	if config.Debug {
		fmt.Println("HelloReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}

func (sched *Scheduler) SendPublicKey(dest *net.UDPAddr) {

	if config.Debug {
		fmt.Println("Sending PublicKey to ", sched.PeerDatabase[dest.String()].Name)
	}

	msg := UDPMessage{
		Id:         uint32(rand.Int31()),
		Type:       PublicKey,
		Length:     64,
		Body:       crypto.FormatPublicKey(*sched.PublicKey),
		PrivateKey: sched.PrivateKey,
	}
	_, err := sched.SendPacket(msg, dest)
	if err != nil {
		fmt.Println("SendPublicKey: ", err.Error())
		return
	}
}

func (sched *Scheduler) SendPublicKeyReply(dest *net.UDPAddr, id uint32) {

	msg := UDPMessage{
		Id:         id,
		Type:       PublicKeyReply,
		Length:     64,
		Body:       crypto.FormatPublicKey(*sched.PublicKey),
		PrivateKey: sched.PrivateKey,
	}
	err := sched.Socket.SendPacket(msg, dest)
	if err != nil {
		fmt.Println("SendPublicKeyReply: ", err.Error())
		return
	}

	if config.Debug {
		fmt.Println("PublicKeyReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}

func (sched *Scheduler) SendRoot(dest *net.UDPAddr) {

	if config.Debug {
		fmt.Println("Sending Root to ", sched.PeerDatabase[dest.String()].Name)
	}

	msg := UDPMessage{
		Id:         uint32(rand.Int31()),
		Type:       Root,
		Length:     32,
		Body:       sched.ExportedFiles.Hash[:],
		PrivateKey: sched.PrivateKey,
	}
	_, err := sched.SendPacket(msg, dest)
	if err != nil {
		fmt.Println("SendRoot: ", err.Error())
		return
	}
}

func (sched *Scheduler) SendRootReply(dest *net.UDPAddr, id uint32) {

	msg := UDPMessage{
		Id:         id,
		Type:       RootReply,
		Length:     32,
		Body:       sched.ExportedFiles.Hash[:],
		PrivateKey: sched.PrivateKey,
	}
	err := sched.Socket.SendPacket(msg, dest)
	if err != nil {
		fmt.Println("SendRootReply: ", err.Error())
		return
	}

	if config.Debug {
		fmt.Println("RootReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}
