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
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var ENDPOINT = "https://jch.irif.fr:8443"

var scheduler *udptypes.Scheduler
var peersNames = widget.NewLabel("")

// taille max d'un chunk en octets
const ChunkSize = 1024

// nombre min de fils d'un bigfile
const MinChildren = 2

// nombre max de fils d'un bigfile
const MaxChildren = 32

// Charge le fichier, à partir du chemin donné, et de ses enfants (si c'est un big file)
func loadFile(path string, name string, data []byte) (filestructure.File, error) {
	if len(data) <= ChunkSize {
		chunk := filestructure.Chunk{
			Name: name,
			Data: data,
		}

		hash := sha256.Sum256(chunk.Data)
		chunk.Hash = hash

		return chunk, nil
	} else {
		bigFile := filestructure.Bigfile{
			Name: name,
		}

		childSize := (len(data) + MaxChildren - 1) / MaxChildren
		if childSize < ChunkSize {
			childSize = ChunkSize
		}

		for i := 0; i < len(data); i += childSize {
			end := i + childSize
			if end > len(data) {
				end = len(data)
			}

			child, err := loadFile(path, name+fmt.Sprintf(" part %d", i/childSize), data[i:end])
			if err != nil {
				return nil, err
			}

			bigFile.Data = append(bigFile.Data, child)
		}

		hash := sha256.Sum256(data)
		bigFile.Hash = hash

		return bigFile, nil
	}
}

// Charge le répertoire à partir du chemin donné et de ses enfants
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

		return loadFile(path, fileInfo.Name(), data)
	}
}

func handleBigfile(bigfile filestructure.Bigfile) ([]byte, error) {
	var data []byte
	for _, child := range bigfile.Data {
		switch child := child.(type) {
		case filestructure.Chunk:
			data = append(data, child.Data...)
		case filestructure.Bigfile:
			childData, err := handleBigfile(child)
			if err != nil {
				return nil, err
			}
			data = append(data, childData...)
		default:
			return nil, fmt.Errorf("unexpected type in Bigfile: %T", child)
		}
	}
	return data, nil
}

func saveFileStructure(path string, node filestructure.File) error {
	fmt.Println("Saving : ", path)
	switch node := node.(type) {
	case filestructure.Chunk:
		return os.WriteFile(path, node.Data, 0644)
	case filestructure.Bigfile:
		data, err := handleBigfile(node)
		if err != nil {
			return err
		}
		return os.WriteFile(path, data, 0644)
	case filestructure.Directory:
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
		for _, child := range node.Data {
			switch child := child.(type) {
			case filestructure.Directory:
				childPath := filepath.Join(path, child.Name)
				if err := saveFileStructure(childPath, child); err != nil {
					return err
				}
			case filestructure.Chunk:
				childPath := filepath.Join(path, child.Name)
				if err := saveFileStructure(childPath, child); err != nil {
					return err
				}
			case filestructure.Bigfile:
				childPath := filepath.Join(path, child.Name)
				if err := saveFileStructure(childPath, child); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unexpected type: %T", child)
			}
		}
	default:
		return fmt.Errorf("unexpected type: %T", node)
	}
	return nil
}

// Print the file structure
func printFileStructure(file filestructure.File, indent string, simplified bool) {
	switch f := file.(type) {
	case filestructure.Directory:
		fmt.Println(indent + f.Name + "/")
		for _, child := range f.Data {
			printFileStructure(child, indent+"  ", simplified)
		}
	case filestructure.Chunk:
		fmt.Println(indent + f.Name)
	case filestructure.Bigfile:
		fmt.Println(indent + f.Name + " (bigfile)")
		if simplified {
			fmt.Println(indent + "  nombre de fils: " + strconv.Itoa(len(f.Data)))
		} else {
			for _, child := range f.Data {
				printFileStructure(child, indent+"  ", simplified)
			}
		}
	default:
		fmt.Println("Unknown file type")
	}
}

func main() {

	file, err := loadDirectory("test_arborescence")
	if err != nil {
		log.Fatal(err)
	} else if config.Debug {
		/* passer true à false pour afficher tous les fichiers (descendants bigfiles)
		 * ATTENTION : risque de faire laguer si l'arborescence est trop grande
		 */
		printFileStructure(file, "", true)
	}
	saveFileStructure("test_arborescence_copy", file)

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

		peer := scheduler.PeerDatabase[serverIp]
		peer.TreeStructure = filestructure.Directory{}

		datumRoot := udptypes.UDPMessage{
			Id:     uint32(rand.Int31()),
			Type:   udptypes.GetDatum,
			Length: 32,
			Body:   peer.Root[:],
		}

		scheduler.Enqueue(datumRoot, ip)

		node := <-scheduler.DatumReceiver
		body := udptypes.BytesToDatumBody(node.Packet.Body)

		for i := 1; i < len(body.Value); i += 64 {
			child := filestructure.Child{
				Name: string(body.Value[i : i+32]),
				Hash: [32]byte(body.Value[i+32 : i+64]),
			}
			peer.TreeStructure.Children = append(peer.TreeStructure.Children, child)
		}

		peer.TreeStructure.Name = peer.Name + "-" + time.Now().Format("2006-01-02_15-04")

		scheduler.DownloadNode((*filestructure.Node)(&peer.TreeStructure), serverIp)
		saveFileStructure(peer.TreeStructure.Name, peer.TreeStructure)

		fmt.Println("\n"+scheduler.PeerDatabase[serverIp].TreeStructure.Name, len(scheduler.PeerDatabase[serverIp].TreeStructure.Data))
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
