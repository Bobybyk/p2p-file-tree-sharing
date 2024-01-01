package udptypes

import (
	"crypto/sha256"
	"fmt"
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
		PeerDatabase:  make(map[string]PeerInfo),
		PacketSender:  make(chan SchedulerEntry),
		DatumReceiver: make(chan SchedulerEntry),
	}
}

func verifyDatumHash(datum DatumBody) bool {
	hash := sha256.Sum256(datum.Value)
	return hash == datum.Hash
}

func (sched *Scheduler) HandleReceive(received UDPMessage, from net.Addr) {

	//register user in the database
	if received.Type == HelloReply || received.Type == Hello {
		body := BytesToHelloBody(received.Body)
		sched.PeerDatabase[from.String()] = PeerInfo{Name: body.Name}
	}

	//if the user is not present in the database, ignore the message as it did not complete handshake
	peer, ok := sched.PeerDatabase[from.String()]
	if !ok {
		if config.Debug {
			fmt.Println("Ignored Message from " + from.String() + " (did not complete handshake)")
		}
		return
	}

	if received.Type == NoOp || ((peer.LastPacketSent == nil || peer.LastPacketSent.Packet.Id != received.Id) && received.Type >= NatTraversalRequest && received.Type != HelloReply) {
		fmt.Println("Unrequested Packet (" + strconv.Itoa(int(received.Type)) + ") from " + peer.Name + " -> throwing it away")
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

		if peerEdit.TreeStructure.Hash != peerEdit.Root {
			peerEdit.TreeStructure = filestructure.Directory{
				Name: peer.Name + "-" + time.Now().Format("2006-01-02_15-04"),
				Hash: peerEdit.Root,
			}
		}

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

			fmt.Println("Datum from: " + peer.Name)
			switch body.Value[0] {
			case 0:
				fmt.Println("Received Chunk")
			case 1:
				fmt.Println("Received BigFile")
				fmt.Println("Number of children: " + (strconv.Itoa((len(body.Value) - 1) / 32)))
			case 2:
				fmt.Println("Received Directory")
				fmt.Println("Number of files: " + (strconv.Itoa((len(body.Value) - 1) / 64)))
				for i := 1; i < len(body.Value); i += 64 {
					fmt.Println(string(body.Value[i : i+32]))
				}
			}
		}
		entry := SchedulerEntry{
			From:   from,
			Time:   time.Now(),
			Packet: received,
		}
		select {
		case sched.DatumReceiver <- entry: //try to send packet to receive packet
		default: // if the reader is busy, get ignore packet
			if config.Debug {
				fmt.Println("Datum received but ignored (busy)")
			}
			break
		}
		peer.LastPacketSent = nil
	case NoDatum:
		//TODO
		if config.Debug {
			fmt.Println("NoDatum from: " + peer.Name)
		}
		peer.LastPacketSent = nil
	default:
		fmt.Println(received.Type)
	}
}

func (sched *Scheduler) DatumReceivePending() {
	for {
		select {
		case datumEntry := <-sched.DatumReceiver:
			datumFrom := datumEntry.From
			peer := sched.PeerDatabase[datumFrom.String()]

			body := BytesToDatumBody(datumEntry.Packet.Body)

			switch body.Value[0] {
			case 0:
				chunk := filestructure.Chunk{
					Hash: body.Hash,
					Data: body.Value[1:],
				}
				peer.TreeStructure.UpdateDirectory(chunk.Hash, chunk)
			case 1:
				bigfile := filestructure.Bigfile{
					Hash: body.Hash,
				}

				for i := 1; i < len(body.Value); i += 32 {
					bigfile.Data = append(bigfile.Data, filestructure.EmptyNode{
						Hash: [32]byte(body.Value[i : i+32]),
					})
				}
				peer.TreeStructure.UpdateDirectory(bigfile.Hash, bigfile)
			case 2:
				dir := filestructure.Directory{
					Hash: body.Hash,
					Data: make([]filestructure.File, 0),
				}
				for i := 1; i < len(body.Value)-1; i += 64 {
					dir.Data = append(dir.Data, filestructure.EmptyNode{
						Name: string(body.Value[i : i+32]),
						Hash: [32]byte(body.Value[i+32 : i+64]),
					})
				}
				peer.TreeStructure.UpdateDirectory(dir.Hash, dir)
			}

			sched.PeerDatabase[datumFrom.String()] = peer
		}
	}
}

func (sched *Scheduler) SendPending(sock *UDPSock) {
	for {
		select {
		case msgToSend := <-sched.PacketSender:
			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil && config.DebugSpam {
				fmt.Println("Message sent on socket")
			}
			if msgToSend.Packet.Type < NatTraversalRequest {

				dest := sched.PeerDatabase[msgToSend.To.String()]
				dest.LastPacketSent = &msgToSend
				sched.PeerDatabase[msgToSend.To.String()] = dest
				if config.DebugSpam {
					fmt.Println("Memorized packet")
				}
			}
		}
	}
}

func (sched *Scheduler) Reissuer(sock *UDPSock) {

	for k, v := range sched.PeerDatabase {

		entry := v.LastPacketSent

		if entry != nil && entry.Time.Add(time.Second).Before(time.Now()) {

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

	go sched.DatumReceivePending()

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
