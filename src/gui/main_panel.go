package gui

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"net"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
)

var peersNames []string

func Init(scheduler *udptypes.Scheduler, ENDPOINT string) fyne.Window {
	appli := app.New()
	window := appli.NewWindow("Peer to peer file transfer")
	window.Resize(fyne.NewSize(848, 480))

	menu := MakeMenu()
	window.SetMainMenu(menu)

	var selectedPeer string
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

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("Hello button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendHello(ip)
	})
	buttonPublicKey := widget.NewButton("PublicKey", func() {

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("PublicKey button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendPublicKey(ip)
	})
	buttonRoot := widget.NewButton("Root", func() {

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("Root button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendRoot(ip)
	})
	buttonNoOp := widget.NewButton("NoOp", func() {

		ipString, err := rest.GetPeerAddresses(ENDPOINT, selectedPeer)
		if err != nil {
			fmt.Println("NoOp button: ", err.Error())
			return
		}
		ip, err := net.ResolveUDPAddr("udp", ipString[0])

		scheduler.SendNoOp(ip)
	})
	buttonDownload := widget.NewButton("Download files", func() {
		fmt.Println("Download Files")
		fmt.Println(selectedPeer)
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
