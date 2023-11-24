package main

import (
	"fmt"
	"log"
	"net"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
)

var ENDPOINT = "https://jch.irif.fr:8443"

func main() {

	peers, err := rest.GetPeersNames(ENDPOINT)
	if err != nil {
		log.Fatal("Fetching peers names: " + err.Error())
	}

	addresses, err := rest.GetPeerAddresses(ENDPOINT, peers[0])
	if err != nil {
		log.Fatal("Fetching peer addresses: " + err.Error())
	}

	socket, err := udptypes.NewUDPSocket()
	if err != nil {
		log.Fatal("Creating udp socket: " + err.Error())
	}

	distantAddr, err := net.ResolveUDPAddr("udp", addresses[0])
	if err != nil {
		log.Fatal("ResolveUDPAddr " + err.Error())
	}

	//Hello + HelloReply
	msgBody := udptypes.HelloBody{
		Extensions: 0,
		Name:       "ogu",
	}.HelloBodyToBytes()

	msg := udptypes.UDPMessage{
		Id:     0,
		Type:   udptypes.Hello,
		Length: uint16(len(msgBody)),
		Body:   msgBody,
	}

	err = socket.SendPacket(distantAddr, msg)
	if err != nil {
		log.Fatal("SendPacket " + err.Error())
	}

	response, err := socket.ReceivePacket()
	if err != nil {
		log.Fatal("ReceivePacket " + err.Error())
	}
	if response.Type != udptypes.HelloReply {
		log.Fatal("Wrong response received")
	}

	//Receive PublicKey
	pack, err := socket.ReceivePacket()
	if err != nil {
		log.Fatal(err)
	}
	if pack.Type != udptypes.PublicKey {
		log.Fatal("Wrong message received")
	}

	publicKeyResponseMessage := udptypes.UDPMessage{
		Id:     pack.Id,
		Type:   udptypes.PublicKeyReply,
		Length: 0,
	}

	err = socket.SendPacket(distantAddr, publicKeyResponseMessage)
	if err != nil {
		log.Fatal("Send Public Key response" + err.Error())
	}

	//Root exchange
	pack, err = socket.ReceivePacket()
	if err != nil {
		log.Fatal(err)
	}
	if pack.Type != udptypes.Root {
		fmt.Println("Wrond message received")
	}

	_ = udptypes.UDPMessage{
		Id:   pack.Id,
		Type: udptypes.RootReply,
	}

}
