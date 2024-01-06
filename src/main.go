package main

import (
	"fmt"
	"log"
	mrand "math/rand"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/crypto"
	"protocoles-internet-2023/filestructure"
	"protocoles-internet-2023/gui"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
	"time"
)

var ENDPOINT = "https://jch.irif.fr:8443"

var scheduler *udptypes.Scheduler

func main() {

	file, err := filestructure.LoadDirectory("test_arborescence")
	if err != nil {
		log.Fatal(err)
	} else if config.Debug {
		/* passer true à false pour afficher tous les fichiers (descendants bigfiles)
		 * ATTENTION : risque de faire laguer si l'arborescence est trop grande
		 */
		filestructure.PrintFileStructure(file, "", true)
	}

	privateKey, publicKey, err := crypto.LoadFromDisk("keys.db")
	if err != nil {
		log.Fatal("Could not load cryptographic keys: ", err.Error())
	}

	socket, err := udptypes.NewUDPSocket()
	if err != nil {
		log.Fatal("NewUDPSocket: " + err.Error())
	}

	exported, ok := file.(filestructure.Directory)
	if !ok {
		log.Fatal("Root is not a directory")
	}

	scheduler = udptypes.NewScheduler(*socket, &exported, privateKey, publicKey)
	go scheduler.Launch(socket)

	window := gui.Init(scheduler, ENDPOINT)

	go func() {
		for range time.Tick(time.Second * 10) {
			gui.RefreshPeersNames(ENDPOINT)
		}
	}()

	go func() {
		for range time.Tick(time.Second * 30) {
			if config.Debug {
				fmt.Println("Sending Hello to server to maintain association")
			}
			HelloToServer()
		}
	}()

	if config.Debug {
		fmt.Println("Sending Hello To Server")
	}
	HelloToServer()

	gui.RefreshPeersNames(ENDPOINT)

	window.ShowAndRun()

}

func HelloToServer() {

	peers, _ := rest.GetPeersNames(ENDPOINT)

	serverIndex := 0
	for i := 0; i < len(peers); i++ {
		if peers[i] == "jch.irif.fr" {
			serverIndex = i
			break
		}
	}

	addresses, err := rest.GetPeerAddresses(ENDPOINT, peers[serverIndex])
	if err != nil {
		log.Println("Fetching peer addresses: " + err.Error())
		return
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
		Id:         uint32(mrand.Int31()),
		Type:       udptypes.Hello,
		Length:     uint16(len(msgBody)),
		Body:       msgBody,
		PrivateKey: scheduler.PrivateKey,
	}

	_, err = scheduler.SendPacket(msg, distantAddr)
	if err != nil {
		fmt.Println("Could not send: ", err.Error())
		return
	}
}
