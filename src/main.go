package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"log"
	"math/rand"
	"net"
	"os"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/rest"
	udptypes "protocoles-internet-2023/udp"
	"time"
)

var ENDPOINT = "https://jch.irif.fr:8443"

var scheduler *udptypes.Scheduler
var peersNames = widget.NewLabel("")

func main() {

	scheduler = udptypes.NewScheduler()
	socket, err := udptypes.NewUDPSocket()
	if err != nil {
		log.Fatal("NewUDPSocket:" + err.Error())
	}
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

		serverIp := ""
		for key, elem := range scheduler.PeerDatabase {
			if elem.Name == "jch.irif.fr" {
				serverIp = key
				break
			}
		}

		ip, _ := net.ResolveUDPAddr("udp", serverIp)
		body := scheduler.PeerDatabase[serverIp].Root

		getdatum := udptypes.UDPMessage{
			Id:     uint32(rand.Int31()),
			Type:   udptypes.GetDatum,
			Length: 32,
			Body:   body[:],
		}

		scheduler.Enqueue(getdatum, ip)
	})

	window.SetContent(container.NewBorder(nil, nil, leftPanel, nil, buttonDownloadServer))

	go func() {
		for range time.Tick(time.Second * 10) {
			refreshPeersNames()
		}
	}()

	go func() {
		for range time.Tick(time.Second * 30) {
			HelloToServer()
		}
	}()

	HelloToServer()

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
		Id:     uint32(rand.Int31()),
		Type:   udptypes.Hello,
		Length: uint16(len(msgBody)),
		Body:   msgBody,
	}

	scheduler.Enqueue(msg, distantAddr)
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
		total += peersNamesList[i] + "\n"
	}

	peersNames.SetText(total)
}
