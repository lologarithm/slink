package automation

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/lologarithm/slink/slinkserv/messages"
)

type MockUser struct {
	alive           bool
	conn            *net.UDPConn
	incoming        chan messages.Packet
	outgoing        chan messages.Packet
	partialMessages map[uint32][]*messages.Multipart
	snakeID         uint32
	startTick       uint32
	startTime       time.Time
}

func NewMockUser() *MockUser {
	return &MockUser{
		alive:           true,
		incoming:        make(chan messages.Packet, 100),
		outgoing:        make(chan messages.Packet, 100),
		partialMessages: map[uint32][]*messages.Multipart{},
	}
}

func CreateAccount(mu *MockUser, user, pass string) {
	packet := messages.NewPacket(messages.CreateAcctMsgType, &messages.CreateAcct{
		Name:     user,
		Password: pass,
	})
	_, err := mu.conn.Write(packet.Pack())
	if err != nil {
		fmt.Printf("Failed to write to connection.")
		fmt.Println(err)
	}
}

func ReadMessages(mu *MockUser) {
	widx := 0
	buf := make([]byte, 1024)
	for mu.alive {
		n, err := mu.conn.Read(buf[widx:])
		if err != nil {
			fmt.Printf("  %d Failed to read from conn: %s\n", mu.snakeID, err)
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
			mu.incoming <- pack
		}
	}
}

func Connect(mu *MockUser) bool {
	ra, err := net.ResolveUDPAddr("udp", "localhost:24816")
	if err != nil {
		fmt.Println(err)
		return false
	}
	mu.conn, err = net.DialUDP("udp", nil, ra)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func RunUser(mu *MockUser, exit chan int) {
	go func() {
		<-exit
		mu.alive = false
	}()

	timeout := time.After(time.Millisecond * time.Duration(rand.Intn(500)+1000))
	for mu.alive {
		select {
		case <-timeout:
			tick := uint32((time.Now().UnixNano() - mu.startTime.UnixNano()) / int64(20*time.Millisecond))
			dir := int16(rand.Intn(2)) - 1
			sendmsg(mu, messages.NewPacket(messages.TurnSnakeMsgType, &messages.TurnSnake{
				TickID:    tick,
				Direction: dir,
			}))
			timeout = time.After(time.Millisecond * time.Duration(rand.Intn(500)+1000))
		case msg := <-mu.incoming:
			ProcessMessage(mu, msg)
		}
	}

	log.Printf("shutting down user.")
	disconn := messages.NewPacket(messages.DisconnectedMsgType, &messages.Disconnected{})
	disb := disconn.Pack()
	mu.conn.Write(disb)
}

func ProcessMessage(mu *MockUser, msg messages.Packet) {
	switch msg.Frame.MsgType {
	case messages.CreateAcctRespMsgType:
		sendmsg(mu, messages.NewPacket(messages.JoinGameMsgType, &messages.JoinGame{}))
	case messages.GameMasterFrameMsgType:
		// tmsg := msg.NetMsg.(*messages.GameMasterFrame)
		// fmt.Printf("\nServer Frame @ %d.\n---------------------------------\n", tmsg.Tick)
		// for _, e := range tmsg.Entities {
		// 	if e.EType == slinkserv.ETypeHead {
		// 		fmt.Printf("Snake %d: @ (%d,%d) Facing: (%d,%d)\n", e.ID, e.X, e.Y, e.Facing.X, e.Facing.Y)
		// 	} else {
		// 		fmt.Printf("Segment: %d @ (%d,%d)\n", e.ID, e.X, e.Y)
		// 	}
		//
		// }
		// fmt.Printf("---------------------------------\n")
	case messages.HeartbeatMsgType:
		mu.conn.Write(msg.Pack())
	case messages.GameConnectedMsgType:
		gcmsg := msg.NetMsg.(*messages.GameConnected)
		mu.snakeID = gcmsg.SnakeID
		mu.startTick = gcmsg.TickID
		mu.startTime = time.Now()
	case messages.SnakeDiedMsgType:
		sdmsg := msg.NetMsg.(*messages.SnakeDied)
		if sdmsg.ID == mu.snakeID {
			os.Exit(0)
		}
	case messages.MultipartMsgType:
		handleMultipart(mu, msg)
	}

}

func sendmsg(mu *MockUser, msg *messages.Packet) {
	_, err := mu.conn.Write(msg.Pack())
	if err != nil {
		fmt.Printf("Failed to write to connection.")
		fmt.Println(err)
	}
}

func handleMultipart(mu *MockUser, packet messages.Packet) {
	netmsg := packet.NetMsg.(*messages.Multipart)
	// 1. Check if this group already exists
	if _, ok := mu.partialMessages[netmsg.GroupID]; !ok {
		mu.partialMessages[netmsg.GroupID] = make([]*messages.Multipart, netmsg.NumParts)
	}
	// 2. Insert into group
	mu.partialMessages[netmsg.GroupID][netmsg.ID] = netmsg
	// 3. See if group is ready to process
	isReady := true
	for _, p := range mu.partialMessages[netmsg.GroupID] {
		if p == nil {
			isReady = false
			break
		}
	}
	if isReady {
		buf := &bytes.Buffer{}
		for _, p := range mu.partialMessages[netmsg.GroupID] {
			buf.Write(p.Content)
		}
		packet, ok := messages.NextPacket(buf.Bytes())
		if !ok {
			fmt.Printf("lol, failed multipart.... %v", packet)
		}
		mu.incoming <- packet
	}
}
