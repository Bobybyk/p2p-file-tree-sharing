package main

import (
	"crypto/ecdsa"
	"fmt"
	"fyne.io/fyne/v2/widget"
	"github.com/rapidloop/skv"
	"log"
	mrand "math/rand"
	"net"
	"os"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/crypto"
	"protocoles-internet-2023/filestructure"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var ENDPOINT = "https://jch.irif.fr:8443"

var scheduler *udptypes.Scheduler
var peersNames = widget.NewLabel("")

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

	var privateKey *ecdsa.PrivateKey
	var publicKey *ecdsa.PublicKey

	//Private and Public key Generation or retrieval from bank
	keysFile, err := skv.Open("keys.db")
	defer keysFile.Close()

	if err == nil {

		var privateKeyString string
		var publicKeyString string

		err := keysFile.Get("private", &privateKeyString)
		if err != nil {
			fmt.Println("could not get private key: ", err.Error())

			privateKey, publicKey, err = crypto.GenerateKeys()
			if err != nil {
				log.Fatal("could not generate keys: ", err.Error())
			}

			privateKeyString, publicKeyString = crypto.EncodeToString(privateKey, publicKey)

			err := keysFile.Put("private", privateKeyString)
			if err != nil {
				log.Fatal("could not store private key: ", err.Error())
			}

			err = keysFile.Put("public", publicKeyString)
			if err != nil {
				log.Fatal("could not store public key: ", err.Error())
			}
		}

		err = keysFile.Get("public", &publicKeyString)
		if err != nil {
			// générer public key
			log.Fatal("could not get public key: ", err.Error())
		}

		privateKey, publicKey = crypto.DecodeFromString(privateKeyString, publicKeyString)
	} /*else {
		privateKey, publicKey, err = crypto.GenerateKeys()
		if err != nil {
			log.Fatal("could not generate keys: ", err.Error())
		}

		err := keysFile.Put("private", privateKey)
		if err != nil {
			log.Fatal("could not store private key: ", err.Error())
		}

		err = keysFile.Put("public", publicKey)
		if err != nil {
			log.Fatal("could not store public key: ", err.Error())
		}
	}*/

	socket, err := udptypes.NewUDPSocket()
	if err != nil {
		log.Fatal("NewUDPSocket:" + err.Error())
	}

	exported, ok := file.(filestructure.Directory)
	if !ok {
		log.Fatal("Root is not a directory")
	}

	scheduler = udptypes.NewScheduler(*socket, &exported, privateKey, publicKey)
	go scheduler.Launch(socket)

	appli := app.New()
	window := appli.NewWindow("Réseau")
	window.Resize(fyne.NewSize(848, 480))

	menu := makeMenu()
	window.SetMainMenu(menu)

	peersNames = widget.NewLabel("")
	refreshPeersButton := widget.NewButton("Refresh", func() {
		refreshPeersNames()
	})
	leftPanel := container.NewBorder(widget.NewLabel("Registered peers"), refreshPeersButton, nil, nil, peersNames)

	buttonDownloadServer := widget.NewButton("Download server", func() {

		peers, err := rest.GetPeersNames(ENDPOINT)
		if err != nil {
			log.Fatal("GET /peers/: " + err.Error())
		}

		peerIndex := 0
		for i := 0; i < len(peers); i++ {
			if peers[i] == "jch.irif.fr" { //change here to change client to download
				peerIndex = i
				break
			}
		}

		addresses, err := rest.GetPeerAddresses(ENDPOINT, peers[peerIndex])
		if err != nil {
			log.Fatal("Fetching peer addresses: " + err.Error())
		}

		msgBody := udptypes.HelloBody{
			Extensions: 0,
			Name:       config.ClientName,
		}.HelloBodyToBytes()

		msg := udptypes.UDPMessage{
			Id:         uint32(mrand.Int31()),
			Type:       udptypes.Hello,
			Length:     uint16(len(msgBody)),
			Body:       msgBody,
			PrivateKey: privateKey,
		}

		ip, _ := net.ResolveUDPAddr("udp", addresses[0])

		_, err = scheduler.SendPacket(msg, ip)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		msg = udptypes.UDPMessage{
			Id:         uint32(mrand.Int31()),
			Type:       udptypes.PublicKey,
			Length:     64,
			Body:       crypto.FormatPublicKey(*scheduler.PublicKey),
			PrivateKey: privateKey,
		}
		_, err = scheduler.SendPacket(msg, ip)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		msg = udptypes.UDPMessage{
			Id:         uint32(mrand.Int31()),
			Type:       udptypes.Root,
			Length:     32,
			Body:       scheduler.ExportedFiles.Hash[:],
			PrivateKey: privateKey,
		}
		_, err = scheduler.SendPacket(msg, ip)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		peer := scheduler.PeerDatabase[ip.String()]
		downloadedNode := &filestructure.Directory{}

		datumRoot := udptypes.UDPMessage{
			Id:     uint32(mrand.Int31()),
			Type:   udptypes.GetDatum,
			Length: 32,
			Body:   peer.Root[:],
		}

		packet, err := scheduler.SendPacket(datumRoot, ip)
		if err != nil {
			log.Fatal("Could not send GetDatum packet: ", err.Error())
		}

		node := packet
		if packet.Packet.Type == udptypes.NoDatum {
			fmt.Println("No datum received")
			return
		}
		body := udptypes.BytesToDatumBody(node.Packet.Body)

		for i := 1; i < len(body.Value); i += 64 {
			child := filestructure.Child{
				Name: string(body.Value[i : i+32]),
				Hash: [32]byte(body.Value[i+32 : i+64]),
			}

			downloadedNode.Children = append(downloadedNode.Children, child)
		}

		downloadedNode.Name = peer.Name + "-" + time.Now().Format("2006-01-02_15-04")

		newNode, err := scheduler.DownloadNode((*filestructure.Node)(downloadedNode), ip.String())
		if err != nil {
			fmt.Println("Download files:", err.Error())
			return
		}

		err = filestructure.SaveFileStructure("../"+newNode.Name, *(*filestructure.Directory)(newNode))
		if err != nil {
			fmt.Println("saving file structure: ", err.Error())
		}

	})

	window.SetContent(container.NewBorder(nil, nil, leftPanel, nil, buttonDownloadServer))

	go func() {
		for range time.Tick(time.Second * 10) {
			refreshPeersNames()
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

	refreshPeersNames()

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
		log.Fatal("Fetching peer addresses: " + err.Error())
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

func makeMenu() *fyne.MainMenu {
	quitItem := fyne.NewMenuItem("Quit", func() {
		os.Exit(0)
	})
	fileCategory := fyne.NewMenu("File", quitItem)

	debugCheckbox := fyne.NewMenuItem("Debug", nil)
	debugCheckbox.Action = func() {
		config.SetDebug(!debugCheckbox.Checked)
		debugCheckbox.Checked = !debugCheckbox.Checked
	}
	debugCheckbox.Checked = true

	extendedDebug := fyne.NewMenuItem("Extended debug", nil)
	extendedDebug.Action = func() {
		config.SetDebugSpam(!extendedDebug.Checked)
		extendedDebug.Checked = !extendedDebug.Checked
	}
	debugCheckbox.Checked = true

	loggingCategory := fyne.NewMenu("Logging", debugCheckbox, extendedDebug)

	return fyne.NewMainMenu(fileCategory, loggingCategory)
}

func refreshPeersNames() {
	peersNamesList, err := rest.GetPeersNames(ENDPOINT)
	if err != nil {
		log.Fatal("Fetching peers names: " + err.Error())
	}

	total := ""

	for i := 0; i < len(peersNamesList); i++ {
		if len(peersNamesList[i]) > 14 {
			peersNamesList[i] = peersNamesList[i][:14]
		}
		total += peersNamesList[i] + "\n"

	}

	peersNames.SetText(total)
}
