package main

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"protocoles-internet-2023/config"
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

// Load a directory recursively
func loadDirectory(path string) (filestructure.File, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if fileInfo.IsDir() {
		node := filestructure.Directory{
			Name: fileInfo.Name(),
		}

		children, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}

		for _, child := range children {
			childFile, err := loadDirectory(filepath.Join(path, child.Name()))
			if err != nil {
				return nil, err
			}
			node.Data = append(node.Data, childFile)
		}

		// Compute the hash of the directory
		hash := sha256.Sum256([]byte(node.Name))
		node.Hash = hash

		return node, nil
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		chunk := filestructure.Chunk{
			Name: fileInfo.Name(),
			Data: data,
		}

		// Compute the hash of the file
		hash := sha256.Sum256(data)
		chunk.Hash = hash

		return chunk, nil
	}
}

// Print the file structure
func printFileStructure(file filestructure.File, indent string) {
	switch f := file.(type) {
	case filestructure.Directory:
		fmt.Println(indent + f.Name + "/")
		for _, child := range f.Data {
			printFileStructure(child, indent+"  ")
		}
	case filestructure.Chunk:
		fmt.Println(indent + f.Name)
	default:
		fmt.Println("Unknown file type")
	}
}

func main() {

	file, err := loadDirectory("test")
	if err != nil {
		log.Fatal(err)
	} else if config.Debug {
		printFileStructure(file, "")
	}

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

		_, _ = net.ResolveUDPAddr("udp", serverIp)

		//TODO get peer's files

		time.Sleep(time.Second * 2)
		fmt.Println("\n\n\n"+scheduler.PeerDatabase[serverIp].TreeStructure.Name, len(scheduler.PeerDatabase[serverIp].TreeStructure.Data))
		for i := 0; i < len(scheduler.PeerDatabase[serverIp].TreeStructure.Data); i++ {
			fmt.Println(scheduler.PeerDatabase[serverIp].TreeStructure.Data[i])
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
		if len(peersNamesList[i]) > 14 {
			peersNamesList[i] = peersNamesList[i][:14]
		}
		total += peersNamesList[i] + "\n"

	}

	peersNames.SetText(total)
}
