package udptypes

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"net"
	"protocoles-internet-2023/config"
	"protocoles-internet-2023/filestructure"
	"strconv"
	"time"
)

/*
Scheduler "constructor"
*/
func NewScheduler() *Scheduler {
	return &Scheduler{
		PeerDatabase:  make(map[string]*PeerInfo),
		PacketSender:  make(chan SchedulerEntry),
		DatumReceiver: make(chan SchedulerEntry),
	}
}

func verifyDatumHash(datum DatumBody) bool {
	hash := sha256.Sum256(datum.Value)
	return hash == datum.Hash
}

func (sched *Scheduler) DownloadNode(node *filestructure.Node, ip string) *filestructure.Node {

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

		sched.Enqueue(getDatum, ipAddr)

		datumEntry := <-sched.DatumReceiver

		body := BytesToDatumBody(datumEntry.Packet.Body)

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
			node.Data[i] = (filestructure.Directory)(*sched.DownloadNode((*filestructure.Node)(&datanode), ip))
		} else if datanode, ok := data.(filestructure.Bigfile); ok {
			node.Data[i] = (filestructure.Bigfile)(*sched.DownloadNode((*filestructure.Node)(&datanode), ip))
		}
	}

	return node
}

func (sched *Scheduler) HandleReceive(received UDPMessage, from net.Addr) {

	//register user in the database
	if received.Type == HelloReply || received.Type == Hello {
		body := BytesToHelloBody(received.Body)
		sched.PeerDatabase[from.String()] = &PeerInfo{
			Name:           body.Name,
			LastPacketSent: new(SchedulerEntry),
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
	/*
		if (peer.LastPacketSent == nil || peer.LastPacketSent.Packet.Id != received.Id) && received.Type >= NatTraversalRequest && received.Type != HelloReply {
			fmt.Println("Unrequested Packet (" + strconv.Itoa(int(received.Type)) + ") from " + peer.Name + " -> throwing it away")
			return
		}*/

	if peer.LastPacketSent == nil && received.Type >= NatTraversalRequest && received.Type != HelloReply {
		fmt.Println("Unrequested Packet (" + strconv.Itoa(int(received.Type)) + ") from " + peer.Name + " (not waiting for response) -> throwing it away")
		return
	} else if (peer.LastPacketSent != nil && peer.LastPacketSent.Packet.Id != received.Id) && received.Type >= NatTraversalRequest && received.Type != HelloReply {
		fmt.Println("Unrequested Packet (" + strconv.Itoa(int(received.Type)) + ") from " + peer.Name + " (wrong ID) -> throwing it away")
		return
	}

	distantPeer, _ := net.ResolveUDPAddr("udp", from.String())

	//otherwise handle the messages
	switch received.Type {
	case NoOp:
		//TODO
		if config.Debug {
			fmt.Println("NoOp from: " + peer.Name)
		}
	case Error:
		//TODO
		if config.Debug {
			fmt.Println("Error from: " + peer.Name)
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
			peerEdit := sched.PeerDatabase[distantPeer.String()]
			peerEdit.PublicKey = received.Body
			sched.PeerDatabase[distantPeer.String()] = peerEdit
		} else {
			peerEdit := sched.PeerDatabase[distantPeer.String()]
			peerEdit.PublicKey = nil
			sched.PeerDatabase[distantPeer.String()] = peerEdit
		}
		sched.SendPublicKeyReply(distantPeer, received.Id)
	case Root:
		if config.Debug {
			fmt.Println("Root from: " + peer.Name)
		}

		peerEdit := sched.PeerDatabase[distantPeer.String()]
		peerEdit.Root = [32]byte(received.Body)

		sched.PeerDatabase[distantPeer.String()] = peerEdit
		sched.SendRootReply(distantPeer, received.Id)
	case GetDatum:

		if config.Debug {
			fmt.Println("Getdatum from: " + peer.Name)
		}

		// reply with the resquested node datum
		peerEdit := sched.PeerDatabase[distantPeer.String()]
		node := peerEdit.TreeStructure.GetNode([32]byte(received.Body))
		if node != nil {
			var nodeBytes []byte
			switch node := node.(type) {
			case filestructure.Chunk:
				nodeBytes = DatumBody{
					Hash:  node.Hash,
					Value: append([]byte{0}, node.Data...),
				}.DatumBodyToBytes()
			case filestructure.Bigfile:
				nodeBytes = DatumBody{
					Hash: node.Hash,
				}.DatumBodyToBytes()
				for _, child := range node.Data {
					hash := child.(filestructure.Node).Hash
					nodeBytes = append(nodeBytes, hash[:]...)
				}
			case filestructure.Directory:
				nodeBytes = DatumBody{
					Hash: node.Hash,
				}.DatumBodyToBytes()
				for _, child := range node.Data {
					nodeBytes = append(nodeBytes, []byte(child.(filestructure.Node).Name)...)
					hash := child.(filestructure.Node).Hash
					nodeBytes = append(nodeBytes, hash[:]...)
				}
			}

			msg := UDPMessage{
				Id:     received.Id,
				Type:   Datum,
				Length: uint16(len(nodeBytes)),
				Body:   nodeBytes,
			}
			sched.Enqueue(msg, distantPeer)
		} else {
			msg := UDPMessage{
				Id:     received.Id,
				Type:   NoDatum,
				Length: 0,
			}
			sched.Enqueue(msg, distantPeer)
		}
	case HelloReply:
		//TODO
		if config.Debug {
			fmt.Println("HelloReply From: " + peer.Name)
		}
		peer.LastPacketSent = nil
	case PublicKeyReply:
		//TODO
		if config.Debug {
			fmt.Println("=============\nPublicKey from: " + peer.Name + "\n=============")
		}
		peer.LastPacketSent = nil
	case RootReply:
		//TODO
		if config.Debug {
			fmt.Println("RootReply from: " + peer.Name)
		}
		peer.LastPacketSent = nil
	case Datum:
		if !verifyDatumHash(BytesToDatumBody(received.Body)) {
			if config.Debug {
				fmt.Println("Invalid hash for datum")
			}
			return
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
		peer.LastPacketSent = nil
		select {
		case sched.DatumReceiver <- entry: //try to send packet to receive packet
		default: // if the reader is busy, get ignore packet
			if config.Debug {
				fmt.Println("Datum received but ignored (busy)")
			}
			break
		}
	case NoDatum:
		//TODO
		if config.Debug {
			fmt.Println("NoDatum from: " + peer.Name)
		}
		peer.LastPacketSent = nil
	default:
		fmt.Println(received.Type, " from: ", peer.Name)
	}
}

func (sched *Scheduler) SendPending(sock *UDPSock) {
	for {
		select {
		case msgToSend := <-sched.PacketSender:
			if msgToSend.Packet.Type < NatTraversalRequest {

				if peer, ok := sched.PeerDatabase[msgToSend.To.String()]; ok {
					peer.LastPacketSent = &msgToSend
				} else {
					fmt.Println("no peer")
				}

				if config.DebugSpam {
					fmt.Println("Memorized packet")
				}
			}

			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil && config.DebugSpam {
				fmt.Println("Message sent on socket")
			}
		}
	}
}

func (sched *Scheduler) Reissuer(sock *UDPSock) {

	for k, v := range sched.PeerDatabase {

		entry := v.LastPacketSent

		if entry.Time.Add(time.Second).Before(time.Now()) {

			if config.Debug {
				fmt.Println("Reissuing packet")
			}

			err := sock.SendPacket(entry.To, entry.Packet)
			if err != nil {
				return
			}
			entry.Time = time.Now()
			sched.PeerDatabase[k] = v
		}
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

	go sched.SendPending(sock)

	//go sched.DatumReceivePending()

	//TODO: refactor -> must not be launched here
	//go sched.Reissuer(sock)
}

/*
This function signals to the Launch function that a packet is waiting to be sent
*/
func (sched *Scheduler) Enqueue(message UDPMessage, dest *net.UDPAddr) {

	entry := SchedulerEntry{
		To:     dest,
		Packet: message,
		Time:   time.Now(),
	}
	sched.PacketSender <- entry

	if config.DebugSpam {
		fmt.Println("Message sent on channel")
	}
}

func (sched *Scheduler) SendHelloReply(dest *net.UDPAddr, id uint32) {
	body := HelloBody{
		Name:       config.ClientName,
		Extensions: 0,
	}.HelloBodyToBytes()
	msg := UDPMessage{
		Id:     id,
		Type:   HelloReply,
		Length: uint16(len(body)),
		Body:   body,
	}
	sched.Enqueue(msg, dest)

	if config.Debug {
		fmt.Println("HelloReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}

/*
Tells the peer that no encryption method is used (hardcoded, to change if encryption is implemented later)
*/
func (sched *Scheduler) SendPublicKeyReply(dest *net.UDPAddr, id uint32) {

	msg := UDPMessage{
		Id:     id,
		Type:   PublicKeyReply,
		Length: 0,
	}
	sched.Enqueue(msg, dest)

	if config.Debug {
		fmt.Println("PublicKeyReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}

func (sched *Scheduler) SendRootReply(dest *net.UDPAddr, id uint32) {

	emptyHash := sha256.Sum256([]byte(""))

	msg := UDPMessage{
		Id:     id,
		Type:   RootReply,
		Length: 32,
		Body:   emptyHash[:],
	}
	sched.Enqueue(msg, dest)

	if config.Debug {
		fmt.Println("RootReply sent to: " + sched.PeerDatabase[dest.String()].Name)
	}
}

/*
Pretty printing for a SchedulerEntry
*/
func (entry SchedulerEntry) String() string {
	return "From: " + entry.From.String() + "\n\t@ " + entry.Time.String() + "\nType:" + strconv.Itoa(int(entry.Packet.Type))
}
