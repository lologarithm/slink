package slinkserv

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
)

func TestBasicServer(t *testing.T) {
	exit := make(chan int, 10)
	complete := make(chan int, 1)
	s := NewServer(exit)
	go RunServer(s, exit, complete)

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
		fmt.Printf("buffer: %d\n", buf[:128])
		fmt.Printf("Incorrect response message! Expected: %d, Actual: %d\n", byte(messages.LoginRespMsgType), buf[0])
		t.FailNow()
	}
	packet = messages.NewPacket(messages.DisconnectedMsgType, &messages.Disconnected{})
	conn.Write(packet.Pack())
	for i := 0; i < 10; i++ {
		exit <- 1
	}
	conn.Close()
	<-complete
	fmt.Printf("TestBasicServer complete.\n")
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
	fakeClient.FromNetwork.Close()
	fmt.Printf("bytes/s processed: %.0f\n", float64(t)/time.Now().Sub(st).Seconds())
}

func TestMultipartMessage(t *testing.T) {
	maxPacketSize = 256 // shrink max size to make test work

	exit := make(chan int, 10)
	complete := make(chan int, 1)
	s := NewServer(exit)
	go RunServer(s, exit, complete)

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

	packet = messages.NewPacket(messages.JoinGameMsgType, &messages.JoinGame{})
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
	testComplete := false
	for !testComplete {
		n, err := clientconn.Read(buf[widx:])
		if err != nil {
			fmt.Printf("Failed to read from conn.")
			fmt.Println(err)
			return
		}
		widx += n
		if widx > 0 {
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
						testComplete = true
						break
					}
				}
			}
		}
	}
	maxPacketSize = 512
	packet = messages.NewPacket(messages.DisconnectedMsgType, &messages.Disconnected{})
	clientconn.Write(packet.Pack())
	for i := 0; i < 10; i++ {
		exit <- 1
	}
	clientconn.Close()
	<-complete

}
