package udptypes

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/crypto"
	"protocoles-internet-2023/filestructure"
	"strconv"
	"sync"
	"time"
)

/*
Scheduler "constructor"
*/
func NewScheduler(sock UDPSock, files *filestructure.Directory, prKey *ecdsa.PrivateKey, pubKey *ecdsa.PublicKey) *Scheduler {
	sched := Scheduler{
		Socket:         sock,
		PeerDatabase:   make(map[string]*PeerInfo),
		PrivateKey:     prKey,
		PublicKey:      pubKey,
		PacketReceiver: make(chan SchedulerEntry),
		ExportedFiles:  files,
		Lock:           sync.Mutex{},
	}

	return &sched
}

func verifyDatumHash(datum DatumBody) bool {
	hash := sha256.Sum256(datum.Value)
	return hash == datum.Hash
}

func (sched *Scheduler) DownloadNode(node *filestructure.Node, ip string) (*filestructure.Node, error) {

	var acc []byte
	for _, child := range node.Children {
		acc = append(acc, child.Hash[:]...)
	}
	b := sha256.Sum256(acc)
	if bytes.Equal(b[:], node.Hash[:]) {
		return nil, errors.New("merkle tree could not verify hash")
	}

	ipAddr, _ := net.ResolveUDPAddr("udp", ip)

	getDatum := UDPMessage{
		Id:     uint32(rand.Int31()),
		Type:   GetDatum,
		Length: 32,
		Body:   node.Hash[:],
	}

	for _, child := range node.Children {
		if config.DebugSpam {
			fmt.Println("Requesting child to insert")
		}

		getDatum.Body = child.Hash[:]

		packet, err := sched.SendPacket(getDatum, ipAddr)
		if err != nil {
			fmt.Println("Downloading node: ", err.Error())
			return nil, errors.New("downloading node")
		}

		body := BytesToDatumBody(packet.Packet.Body)

		switch body.Value[0] {
		case 0: //chunk
			newChunk := filestructure.Chunk{
				Data: body.Value[1:],
				Hash: body.Hash,
			}
			if child.Name != "" {
				newChunk.Name = child.Name
			}

			node.Data = append(node.Data, newChunk)
		case 1: //bigfile

			newBig := filestructure.Bigfile{
				Hash: body.Hash,
			}

			if child.Name != "" {
				newBig.Name = child.Name
			}

			for i := 1; i < len(body.Value); i += 32 {
				newBig.Children = append(newBig.Children, filestructure.Child{
					Hash: [32]byte(body.Value[i : i+32]),
				})
			}

			node.Data = append(node.Data, newBig)
		case 2: //directory

			newDir := filestructure.Directory{
				Name: child.Name,
				Hash: body.Hash,
			}
			for i := 1; i < len(body.Value); i += 64 {
				newDir.Children = append(newDir.Children, filestructure.Child{
					Name: string(body.Value[i : i+32]),
					Hash: [32]byte(body.Value[i+32 : i+64]),
				})
			}
			node.Data = append(node.Data, newDir)
		}
	}

	for i, data := range node.Data {
		if datanode, ok := data.(filestructure.Directory); ok {
			nodeTmp, err := sched.DownloadNode((*filestructure.Node)(&datanode), ip)
			if err != nil {
				fmt.Println("downloading child Directory: ", err.Error())
				return nil, errors.New("downloading child directory")
			}
			node.Data[i] = (filestructure.Directory)(*nodeTmp)
		} else if datanode, ok := data.(filestructure.Bigfile); ok {
			nodeTmp, err := sched.DownloadNode((*filestructure.Node)(&datanode), ip)
			if err != nil {
				fmt.Println("downloading child bigfile: ", err.Error())
				return nil, errors.New("downloading child bigfile")
			}
			node.Data[i] = (filestructure.Bigfile)(*nodeTmp)
		}
	}

	return node, nil
}

func (sched *Scheduler) HandleReceive(received UDPMessage, from net.Addr) {

	//register user in the database
	if received.Type == HelloReply || received.Type == Hello {
		body := BytesToHelloBody(received.Body)
		if _, ok := sched.PeerDatabase[from.String()]; !ok {
			sched.PeerDatabase[from.String()] = &PeerInfo{
				Name: body.Name,
			}
		}
	}

	//if the user is not present in the database, ignore the message as it did not complete handshake
	peer, ok := sched.PeerDatabase[from.String()]
	if !ok {
		if config.Debug {
			fmt.Println("Ignored Message from " + from.String() + " (did not complete handshake)")
		}
		return
	}

	if len(peer.PublicKey) > 0 && len(received.Signature) > 0 {
		peerPuKey := crypto.ParsePublicKey(peer.PublicKey)
		if crypto.VerifyMessage(received.MessageToBytes()[:7+received.Length], received.Signature, &peerPuKey) == false {
			fmt.Println("\n", received.Type, " Wrong packet signature")
			return
		}
	}

	distantPeer, _ := net.ResolveUDPAddr("udp", from.String())

	//otherwise handle the messages
	switch received.Type {
	case NoOp:
		if config.Debug {
			fmt.Println("NoOp from: " + peer.Name)
		}
	case Error:
		if config.Debug {
			fmt.Println("Error from: ", peer.Name, "\t", string(received.Body))
		}
	case Hello:
		if config.Debug {
			fmt.Println("Hello from: " + peer.Name)
		}

		sched.SendHelloReply(distantPeer, received.Id)
	case PublicKey:
		if config.Debug {
			fmt.Println("PublicKey from: " + peer.Name)
		}

		if received.Length != 0 {
			peer.PublicKey = received.Body
		} else {
			peer.PublicKey = nil
		}
		sched.SendPublicKeyReply(distantPeer, received.Id)
	case Root:
		if config.Debug {
			fmt.Println("Root from: " + peer.Name)
		}

		peer.Root = [32]byte(received.Body)
		sched.SendRootReply(distantPeer, received.Id)
	case GetDatum:

		if config.Debug {
			fmt.Println("Getdatum from: " + peer.Name)
		}

		// reply with the resquested node datum
		node := (*filestructure.Node)(sched.ExportedFiles).GetNode([32]byte(received.Body))

		if node != nil {
			var nodeBytes []byte
			switch convNode := node.(type) {
			case filestructure.Chunk:
				tmp := DatumBody{
					Value: append([]byte{0}, convNode.Data...),
				}
				tmp.Hash = sha256.Sum256(tmp.Value[:])
				nodeBytes = tmp.DatumBodyToBytes()
			case filestructure.Bigfile:
				tmp := DatumBody{
					Value: []byte{1},
				}
				for _, child := range convNode.Data {

					if ch, ok := child.(filestructure.Chunk); ok {
						tmp.Value = append(tmp.Value, ch.Hash[:]...)
					} else if big, ok := child.(filestructure.Bigfile); ok {
						tmp.Value = append(tmp.Value, big.Hash[:]...)
					}
				}

				tmp.Hash = sha256.Sum256(tmp.Value[:])
				nodeBytes = tmp.DatumBodyToBytes()

			case filestructure.Directory:
				tmp := DatumBody{
					Value: []byte{2},
				}

				for _, child := range convNode.Data {
					if dir, ok := child.(filestructure.Directory); ok {
						tmp.Value = append(tmp.Value, []byte(filestructure.ExpandString(dir.Name))...)
						tmp.Value = append(tmp.Value, dir.Hash[:]...)
					} else if ch, ok := child.(filestructure.Chunk); ok {
						tmp.Value = append(tmp.Value, []byte(filestructure.ExpandString(ch.Name))...)
						tmp.Value = append(tmp.Value, ch.Hash[:]...)
					} else if big, ok := child.(filestructure.Bigfile); ok {
						tmp.Value = append(tmp.Value, []byte(filestructure.ExpandString(big.Name))...)
						tmp.Value = append(tmp.Value, big.Hash[:]...)
					}
				}
				tmp.Hash = sha256.Sum256(tmp.Value[:])

				nodeBytes = tmp.DatumBodyToBytes()
			}

			msg := UDPMessage{
				Id:     received.Id,
				Type:   Datum,
				Length: uint16(len(nodeBytes)),
				Body:   nodeBytes,
			}

			err := sched.Socket.SendPacket(msg, distantPeer)
			if err != nil {
				fmt.Println("Respond Datum: ", err.Error())
				return
			}
		} else {
			msg := UDPMessage{
				Id:     received.Id,
				Type:   NoDatum,
				Length: 0,
			}
			err := sched.Socket.SendPacket(msg, distantPeer)
			if err != nil {
				fmt.Println("Respond no datum: ", err.Error())
				return
			}
		}
	case HelloReply:
		if config.Debug {
			fmt.Println("HelloReply From: " + peer.Name)
		}
		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		sched.PacketReceiver <- entry
	case PublicKeyReply:
		if config.Debug {
			fmt.Println("PublicKeyReply from: " + peer.Name)
		}

		if received.Length != 0 {
			peer.PublicKey = received.Body
		} else {
			peer.PublicKey = nil
		}

		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		sched.PacketReceiver <- entry
	case RootReply:
		if config.Debug {
			fmt.Println("RootReply from: " + peer.Name)
			emptyHash := sha256.Sum256([]byte(""))
			if bytes.Equal(emptyHash[:], received.Body) {
				fmt.Println("The peer does not export any files")
			}
		}
		sched.PeerDatabase[distantPeer.String()].Root = [32]byte(received.Body)
		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		sched.PacketReceiver <- entry
	case Datum:
		if !verifyDatumHash(BytesToDatumBody(received.Body)) {
			if config.Debug {
				fmt.Println("Invalid hash for datum")
			}
		}
		if config.Debug {
			body := BytesToDatumBody(received.Body)

			fmt.Println("\nDatum from: " + peer.Name)
			switch body.Value[0] {
			case 0:
				fmt.Println("Received Chunk")
			case 1:
				fmt.Println("Received BigFile")
				fmt.Println("Number of children: " + (strconv.Itoa((len(body.Value) - 1) / 32)))
			case 2:
				fmt.Println("Received Directory")
				fmt.Println("Number of files: ", (len(body.Value)-1)/64)
				for i := 1; i < len(body.Value); i += 64 {
					fmt.Println(string(body.Value[i : i+32]))
				}
				fmt.Println()
			}
		}
		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		sched.PacketReceiver <- entry
	case NoDatum:
		if config.Debug {
			fmt.Println("NoDatum from: " + peer.Name)
		}
		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		sched.PacketReceiver <- entry

	case ErrorReply:
		if config.Debug {
			fmt.Println("ErrorReply from: " + peer.Name)
			fmt.Println(string(received.Body))
		}
	default:
		fmt.Println(received.Type, " from: ", peer.Name)
	}
}

func (sched *Scheduler) ReceivePending(sock *UDPSock) {
	for {
		received, from, err := sock.ReceivePacket()
		if err != nil {
			//TODO handle
			fmt.Println("error receiving")
		}
		sched.HandleReceive(received, from)
	}
}

/*
	This function manages all I/O on the socket

It automatically receives packets, performs a treatment then sends all pending packets
*/
func (sched *Scheduler) Launch(sock *UDPSock) {
	if config.DebugSpam {
		fmt.Println("Launching scheduler")
	}

	go sched.ReceivePending(sock)
}

/*
This function signals to the Launch function that a packet is waiting to be sent
*/
func (sched *Scheduler) SendPacket(message UDPMessage, dest *net.UDPAddr) (SchedulerEntry, error) {

	sched.Lock.Lock()
	defer sched.Lock.Unlock()

	timeout := 1
	for i := 0; i < 3; i++ {

		sendTime := time.Now()
		err := sched.Socket.SendPacket(message, dest)
		if err == nil && config.DebugSpam {
			fmt.Println("Message sent on socket")
		}

		select {
		case response := <-sched.PacketReceiver:
			if response.Packet.Id != message.Id {
				fmt.Println("wrong response ID\nExpected: " + strconv.Itoa(int(message.Id)) + "\nReceived: " + strconv.Itoa(int(response.Packet.Id)))
			} else {
				receivedTime := time.Now()
				sched.PeerDatabase[dest.String()].RTT = receivedTime.Sub(sendTime).Milliseconds()
				return response, nil
			}
		case <-time.After(time.Second * time.Duration(timeout)):
			if config.Debug {
				fmt.Println("Packet lost -> reemitting")
			}
			timeout *= 2
		}
	}
	return SchedulerEntry{}, errors.New("no response")
}

/*
Pretty printing for a SchedulerEntry
*/
func (entry SchedulerEntry) String() string {
	return "From: " + entry.From.String() + "\n\t@ " + entry.Time.String() + "\nType:" + strconv.Itoa(int(entry.Packet.Type))
}
