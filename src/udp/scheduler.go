package udptypes

import (
	"fmt"
	"net"
	"protocoles-internet-2023/config"
	"time"
)

func NewScheduler() *Scheduler {
	return &Scheduler{
		PacketReceiver: make(chan SchedulerEntry),
		PacketSender:   make(chan SchedulerEntry),
	}
}

func (sched *Scheduler) Launch(sock *UDPSock) {
	if config.DebugSpam {
		fmt.Println("Launching scheduler")
	}

	go sched.HandleReceive()

	for {
		received, _, timeout := sock.ReceivePacket(time.Second * 1)
		if !timeout {
			entry := SchedulerEntry{
				Packet: received,
			}
			sched.PacketReceiver <- entry
		} else if timeout && config.DebugSpam {
			fmt.Println("UDP Receive timeout")
		}

		select {
		case msgToSend := <-sched.PacketSender:
			err := sock.SendPacket(msgToSend.To, msgToSend.Packet)
			if err == nil {
				sched.Lock.Lock()
				//sched.Sent = append(sched.Sent, msgToSend)
				sched.Lock.Unlock()

				if config.Debug {
					fmt.Println("Message received from socket")
				}

			}

			if config.Debug {
				fmt.Println("Message sent on socket")
			}
		default:
			if config.DebugSpam {
				fmt.Println("Nothing to send on socket")
			}
		}
	}

}

func (sched *Scheduler) Enqueue(message UDPMessage, dest *net.UDPAddr) {
	entry := SchedulerEntry{
		To:     dest,
		Packet: message,
	}
	sched.PacketSender <- entry

	if config.Debug {
		fmt.Println("Message sent on channel")
	}
}

func (sched *Scheduler) HandleReceive() {
	if config.Debug {
		fmt.Println("Launched receiver")
	}

	for {
		received := <-sched.PacketReceiver

		if config.Debug {
			fmt.Println("Message received from channel")
		}

		sched.Lock.Lock()
		//sched.Received = append(sched.Received, received)
		sched.Lock.Unlock()
		fmt.Println(received.Packet.Type)
	}
}
