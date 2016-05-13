package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"time"

	"github.com/lologarithm/slink/slinkserv/automation"
)

var numClient = 500

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	exit := make(chan int, numClient)
	for i := 0; i < numClient; i++ {
		mu := automation.NewMockUser()
		connected := automation.Connect(mu)

		if connected {
			automation.CreateAccount(mu, "testuser"+strconv.Itoa(i), "testuser"+strconv.Itoa(i))
			go automation.ReadMessages(mu)
			go automation.RunUser(mu, exit)
		} else {
			c <- os.Interrupt
		}
		time.Sleep(time.Millisecond * 10)
	}

	log.Printf("%d clients started.", numClient)
	<-c

	for i := 0; i < numClient; i++ {
		exit <- i
	}
	time.Sleep(time.Second)
	log.Printf("Goodbye!")
}

//
// func sendMessages(exit chan int) {
// 	ra, err := net.ResolveUDPAddr("udp", "localhost:24816")
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	conn, err := net.DialUDP("udp", nil, ra)
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
//
// 	packet := messages.NewPacket(messages.LoginMsgType, &messages.Login{
// 		Name:     "testuser",
// 		Password: "testpass",
// 	})
// 	msgbytes := packet.Pack()
// 	_, err = conn.Write(msgbytes)
//
// 	alive := true
// 	go func() {
// 		widx := 0
// 		buf := make([]byte, 1024)
// 		for alive {
// 			n, err := conn.Read(buf[widx:])
// 			if err != nil {
// 				fmt.Printf("Failed to read from conn.")
// 				fmt.Println(err)
// 				return
// 			}
// 			widx += n
// 			for {
// 				pack, ok := messages.NextPacket(buf[:widx])
// 				if !ok {
// 					break
// 				}
// 				copy(buf, buf[pack.Len():])
// 				widx -= pack.Len()
// 			}
// 		}
// 	}()
//
// 	go func() {
// 		<-exit
// 		alive = false
// 	}()
//
// 	for alive {
// 		_, err = conn.Write(msgbytes)
// 		if err != nil {
// 			fmt.Printf("Failed to write to connection.")
// 			fmt.Println(err)
// 		}
// 		time.Sleep(time.Millisecond * 100)
// 	}
// 	disconn := messages.NewPacket(messages.DisconnectedMsgType, &messages.Disconnected{})
// 	disb := disconn.Pack()
// 	conn.Write(disb)
// }
