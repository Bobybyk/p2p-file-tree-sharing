package udptypes

import (
	"fmt"
	"net"
	"protocoles-internet-2023/config"
	"strconv"
	"time"
)

/*
Scheduler "constructor"
*/
func NewScheduler() *Scheduler {
	return &Scheduler{
		PeerDatabase: make(map[string]PeerInfo),
		PacketSender: make(chan SchedulerEntry),
	}
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
		sched.SendHelloReply(distantPeer, received.Id)
		if config.Debug {
			fmt.Println("Hello from: " + peer.Name)
		}
	case PublicKey:
		sched.SendPublicKeyReply(distantPeer, received.Id)
		if config.Debug {
			fmt.Println("PublicKey from: " + peer.Name)
		}
	case Root:
		//TODO
		if config.Debug {
			fmt.Println("Root from: " + peer.Name)
		}
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
			fmt.Println("PublicKey from: " + peer.Name)
		}
	case RootReply:
		//TODO
		if config.Debug {
			fmt.Println("RootReply from: " + peer.Name)
		}
	case Datum:
		//TODO
		if config.Debug {
			fmt.Println("Datum from: " + peer.Name)
		}
	default:
		fmt.Println(received.Type)
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

	for {
		received, from, _, timeout := sock.ReceivePacket(time.Millisecond * 100)
		if !timeout {
			sched.HandleReceive(received, from)
		} else if timeout && config.DebugSpam {
			fmt.Println("UDP Receive timeout")
		}

		select {
		case msgToSend := <-sched.PacketSender:
			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil {
				if config.Debug {
					fmt.Println("Message sent on socket")
				}
			}
		default: //if there is nothing to read yet, do not block
			if config.DebugSpam {
				fmt.Println("Nothing to send on socket")
			}
		}
	}

}

/*
This function signals to the Launch function that a packet is waiting to be sent
*/
func (sched *Scheduler) Enqueue(message UDPMessage, dest *net.UDPAddr) {

	//Must be launched in a goroutine to allow Launch to get back to reading on channel
	go func() {
		entry := SchedulerEntry{
			To:     dest,
			Packet: message,
		}
		sched.PacketSender <- entry

		if config.Debug {
			fmt.Println("Message sent on channel")
		}
	}()
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

/*
Pretty printing for a SchedulerEntry
*/
func (entry SchedulerEntry) String() string {
	return "From: " + entry.From.String() + "\n\t@ " + entry.Time.String() + "\nType:" + strconv.Itoa(int(entry.Packet.Type))
}
