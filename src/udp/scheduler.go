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
		PeerDatabase:  make(map[string]*PeerInfo),
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
		sched.PeerDatabase[from.String()] = &PeerInfo{
			Name: body.Name,
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

	if received.Type == NoOp {
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

		sched.PeerDatabase[distantPeer.String()] = peerEdit
		sched.SendRootReply(distantPeer, received.Id)
	case GetDatum:
		//TODO
		if config.Debug {
			fmt.Println("Getdatum from: " + peer.Name)
		}
	case HelloReply:
		//TODO
		if config.Debug {
			fmt.Println("HelloReply From: " + peer.Name)
		}
	case PublicKeyReply:
		//TODO
		if config.Debug {
			fmt.Println("=============\nPublicKey from: " + peer.Name + "\n=============")
		}
	case RootReply:
		//TODO
		if config.Debug {
			fmt.Println("RootReply from: " + peer.Name)
		}
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
	default:
		fmt.Println(received.Type)
	}
}

func (sched *Scheduler) DatumReceivePending() {
	datumEntry := <-sched.DatumReceiver
	datumFrom := datumEntry.From

	body := BytesToDatumBody(datumEntry.Packet.Body)
	parent, name, _ := sched.PeerDatabase[datumFrom.String()].TreeStructure.GetParentNode(body.Hash)

	switch body.Value[0] {
	case 0: //chunk
		newChunk := filestructure.Chunk{
			Data: body.Value[1:],
			Hash: body.Hash,
		}
		if name != "" {
			newChunk.Name = name
		}

		parent.Data = append(parent.Data, newChunk)
	case 1: //bigfile

		newBig := filestructure.Bigfile{
			Hash: body.Hash,
		}

		if name != "" {
			newBig.Name = name
		}

		for i := 1; i < len(body.Value); i += 32 {
			newBig.Children = append(newBig.Children, filestructure.Child{
				Hash: [32]byte(body.Value[i : i+32]),
			})
		}

		parent.Data = append(parent.Data, newBig)
	case 2: //directory

		newDir := filestructure.Directory{
			Name: sched.PeerDatabase[datumFrom.String()].Name + "-" + time.Now().Format("2006-01-02_15-04"),
			Hash: sched.PeerDatabase[datumFrom.String()].Root,
		}
		for i := 1; i < len(body.Value); i += 64 {
			newDir.Children = append(newDir.Children, filestructure.Child{
				Name: string(body.Value[i : i+32]),
				Hash: [32]byte(body.Value[i+32 : i+64]),
			})
		}

		if sched.PeerDatabase[datumFrom.String()].TreeStructure.Hash != sched.PeerDatabase[datumFrom.String()].Root { //receiving root
			if config.DebugSpam {
				fmt.Println("Updating root of ", sched.PeerDatabase[datumFrom.String()].Name)
			}

			sched.PeerDatabase[datumFrom.String()].TreeStructure = newDir
		} else { //insert into tree
			if name != "" {
				newDir.Name = name
			}
			parent.Data = append(parent.Data, newDir)
		}
	}
}

func (sched *Scheduler) SendPending(sock *UDPSock) {
	var msgToSend SchedulerEntry
	for {
		select {
		case msgToSend = <-sched.PacketSender:
			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil && config.DebugSpam {
				fmt.Println("Message sent on socket")
			}
			if msgToSend.Packet.Type < NatTraversalRequest {

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
