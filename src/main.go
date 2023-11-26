package main

import (
	"log"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
	"time"
)

var ENDPOINT = "https://jch.irif.fr:8443"

func main() {

	peers, err := rest.GetPeersNames(ENDPOINT)
	if err != nil {
		log.Fatal("Fetching peers names: " + err.Error())
	}

	serverIndex := 0
	for i := 0; i < len(peers); i++ {
		if peers[i] == "jch.irif.fr" {
			serverIndex = i
			break
		}
	}

	addresses, err := rest.GetPeerAddresses(ENDPOINT, peers[serverIndex])
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
		Name:       config.ClientName,
	}.HelloBodyToBytes()

	msg := udptypes.UDPMessage{
		Id:     120984,
		Type:   udptypes.Hello,
		Length: uint16(len(msgBody)),
		Body:   msgBody,
	}

	scheduler := udptypes.NewScheduler()

	go scheduler.Launch(socket)
	scheduler.Enqueue(msg, distantAddr)
	time.Sleep(time.Second * 1000)
}
