package udptypes

import (
	"fmt"
	"net"
	"time"
)

func NewScheduler() *Scheduler {
	return &Scheduler{
		PacketReceiver: make(chan SchedulerEntry),
		PacketSender:   make(chan SchedulerEntry),
	}
}

func (sched *Scheduler) Launch(sock *UDPSock) {

	go sched.HandleReceive()

	for {
		received, _, timeout := sock.ReceivePacket(time.Second * 1)
		if !timeout {
			entry := SchedulerEntry{
				Packet: received,
			}
			sched.PacketReceiver <- entry
		}

		select {
		case msgToSend := <-sched.PacketSender:
			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil {
				/*sched.Lock.Lock()
				sched.Sent = append(sched.Sent, msgToSend)
				sched.Lock.Unlock()*/
			}
		default:
		}
	}

}

func (sched *Scheduler) Enqueue(message UDPMessage, dest *net.UDPAddr) {
	entry := SchedulerEntry{
		To:     dest,
		Packet: message,
	}
	sched.PacketSender <- entry
}

func (sched *Scheduler) HandleReceive() {
	for {
		received := <-sched.PacketReceiver /*
			sched.Lock.Lock()
			sched.Received = append(sched.Received, received)
			sched.Lock.Unlock()*/
		fmt.Println(received.Packet.Type)
	}
}
