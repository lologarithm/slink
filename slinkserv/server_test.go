package slinkserv

import (
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
)

func TestBasicServer(t *testing.T) {
	exit := make(chan int, 10)
	s := NewServer(exit)
	go RunServer(s, exit)

	time.Sleep(time.Millisecond * 100)
	ra, err := net.ResolveUDPAddr("udp", "localhost:24816")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	conn, err := net.DialUDP("udp", nil, ra)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	packet := messages.NewPacket(messages.LoginMsgType, &messages.Login{
		Name:     "testuser",
		Password: "testpass",
	})
	msgBytes := packet.Pack()
	_, err = conn.Write(msgBytes)
	if err != nil {
		fmt.Printf("Failed to write to connection.")
		fmt.Println(err)
		t.FailNow()
	}
	buf := make([]byte, 512)
	n, err := conn.Read(buf[0:])
	if err != nil {
		fmt.Printf("Failed to read from conn.")
		fmt.Println(err)
		t.FailNow()
	}
	if n < 5 || buf[0] != byte(messages.LoginRespMsgType) {
		fmt.Printf("Incorrect response message!")
		t.FailNow()
	}
	packet = messages.NewPacket(messages.DisconnectedMsgType, &messages.Disconnected{})
	conn.Write(packet.Pack())
	exit <- 1
	conn.Close()
}

func BenchmarkServerParsing(b *testing.B) {
	gamechan := make(chan GameMessage, 100)
	donechan := make(chan Client, 1)
	fakeClient := &Client{
		address:         &net.UDPAddr{},
		FromNetwork:     NewBytePipe(0),
		FromGameManager: make(chan InternalMessage, 10),
		toGameManager:   gamechan,
		ID:              1,
	}
	go fakeClient.ProcessBytes(donechan)

	packet := messages.NewPacket(messages.LoginMsgType, &messages.Login{
		Name:     "testuser",
		Password: "testpass",
	})
	msgBytes := packet.Pack()
	st := time.Now()

	b.ResetTimer()
	t := 0
	for i := 0; i < b.N; i++ {
		fakeClient.FromNetwork.Write(msgBytes)
		<-gamechan
		t += len(msgBytes)
	}
	b.StopTimer()
	log.Printf("bytes/s processed: %.0f", float64(t)/time.Now().Sub(st).Seconds())
}

func TestMultipartMessage(t *testing.T) {
	exit := make(chan int, 10)
	s := NewServer(exit)
	go RunServer(s, exit)

	time.Sleep(time.Millisecond * 100)
	ra, err := net.ResolveUDPAddr("udp", "localhost:24816")
	if err != nil {
		fmt.Println(err)
		t.FailNow()
		return
	}
	clientconn, err := net.DialUDP("udp", nil, ra)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
		return
	}

	log.Printf("Opened client conn!")
	packet := messages.NewPacket(messages.CreateAcctMsgType, &messages.CreateAcct{
		Name:     "testuser",
		Password: "testpass",
	})
	msgbytes := packet.Pack()
	_, err = clientconn.Write(msgbytes)
	if err != nil {
		fmt.Printf("Failed to write to connection.")
		fmt.Println(err)
	}
	time.Sleep(time.Millisecond * 100)

	packet = messages.NewPacket(messages.CreateGameMsgType, &messages.CreateGame{
		Name: "testgame",
	})
	msgbytes = packet.Pack()
	_, err = clientconn.Write(msgbytes)
	if err != nil {
		fmt.Printf("Failed to write to connection.")
		fmt.Println(err)
	}

	time.Sleep(time.Millisecond * 100)

	multibuffer := make([]byte, 0)
	widx := 0
	buf := make([]byte, 8092)
	for {
		n, err := clientconn.Read(buf[widx:])
		if err != nil {
			fmt.Printf("Failed to read from conn.")
			fmt.Println(err)
			return
		}
		widx += n
		for {
			pack, ok := messages.NextPacket(buf[:widx])
			if !ok {
				break
			}
			copy(buf, buf[pack.Len():])
			widx -= pack.Len()
			tmsg, ok := pack.NetMsg.(*messages.Multipart)
			if ok {
				multibuffer = append(multibuffer, tmsg.Content...)
				if tmsg.ID == tmsg.NumParts-1 {
					log.Printf("Reassembled the multi-message successfully.")
					return
				}
			}
		}
	}
}
