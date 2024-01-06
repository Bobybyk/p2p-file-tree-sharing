package gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"log"
	mrand "math/rand"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/crypto"
	"protocoles-internet-2023/filestructure"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
	"time"
)

var peersNames []string

func Init(scheduler *udptypes.Scheduler, ENDPOINT string) fyne.Window {
	appli := app.New()
	window := appli.NewWindow("Peer to peer file transfer")
	window.Resize(fyne.NewSize(848, 480))

	menu := MakeMenu()
	window.SetMainMenu(menu)

	selectedPeer := ""
	peerNamesListWidget := widget.NewList(
		func() int {
			return len(peersNames)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(peersNames[i])
		},
	)
	peerNamesListWidget.OnSelected = func(id widget.ListItemID) {
		selectedPeer = peersNames[id]
	}

	refreshPeersButton := widget.NewButton("Refresh", func() {
		RefreshPeersNames(ENDPOINT)
	})

	leftPanel := container.NewBorder(widget.NewLabel("Registered peers"), refreshPeersButton, nil, nil, peerNamesListWidget)

	buttonHello := widget.NewButton("Hello", func() {

		if selectedPeer == "" {
			fmt.Println("no peer selected")
			return
		}

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("Hello button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendHello(ip)
	})
	buttonPublicKey := widget.NewButton("PublicKey", func() {

		if selectedPeer == "" {
			fmt.Println("no peer selected")
			return
		}

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("PublicKey button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendPublicKey(ip)
	})
	buttonRoot := widget.NewButton("Root", func() {

		if selectedPeer == "" {
			fmt.Println("no peer selected")
			return
		}

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("Root button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendRoot(ip)
	})
	buttonNoOp := widget.NewButton("NoOp", func() {

		if selectedPeer == "" {
			fmt.Println("no peer selected")
			return
		}

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("NoOp button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendNoOp(ip)
	})
	buttonDownload := widget.NewButton("Download files", func() {
		if selectedPeer == "" {
			fmt.Println("no peer selected")
			return
		}

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("NoOp button: ", err.Error())
			return
		}
		peerIP, err := net.ResolveUDPAddr("udp", ipString[0])

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

		_, err = scheduler.SendPacket(msg, peerIP)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		msg = udptypes.UDPMessage{
			Id:         uint32(mrand.Int31()),
			Type:       udptypes.PublicKey,
			Length:     64,
			Body:       crypto.FormatPublicKey(*scheduler.PublicKey),
			PrivateKey: scheduler.PrivateKey,
		}
		_, err = scheduler.SendPacket(msg, peerIP)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		msg = udptypes.UDPMessage{
			Id:         uint32(mrand.Int31()),
			Type:       udptypes.Root,
			Length:     32,
			Body:       scheduler.ExportedFiles.Hash[:],
			PrivateKey: scheduler.PrivateKey,
		}
		_, err = scheduler.SendPacket(msg, peerIP)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		peer := scheduler.PeerDatabase[peerIP.String()]
		downloadedNode := &filestructure.Directory{}

		datumRoot := udptypes.UDPMessage{
			Id:     uint32(mrand.Int31()),
			Type:   udptypes.GetDatum,
			Length: 32,
			Body:   peer.Root[:],
		}

		packet, err := scheduler.SendPacket(datumRoot, peerIP)
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

		newNode, err := scheduler.DownloadNode((*filestructure.Node)(downloadedNode), peerIP.String())
		if err != nil {
			fmt.Println("Download files:", err.Error())
			return
		}

		err = filestructure.SaveFileStructure("../"+newNode.Name, *(*filestructure.Directory)(newNode))
		if err != nil {
			fmt.Println("saving file structure: ", err.Error())
		}

	})

	vboxButtons := container.New(layout.NewVBoxLayout(), buttonHello, buttonRoot, buttonNoOp, buttonPublicKey, buttonDownload)

	buttonArea := container.NewBorder(widget.NewLabel("Actions"), nil, nil, nil, vboxButtons)

	window.SetContent(container.NewBorder(nil, nil, leftPanel, nil, buttonArea))

	return window
}

func RefreshPeersNames(ENDPOINT string) {
	peersNamesList, err := rest.GetPeersNames(ENDPOINT)
	if err != nil {
		fmt.Println("Fetching peers names: " + err.Error())
	}

	peersNames = peersNamesList
}
