package messages

import (
	"bytes"
	"encoding/binary"
	"log"
)

type Net interface {
	Serialize(*bytes.Buffer)
	Deserialize(*bytes.Buffer)
	Len() int
}

type MessageType uint16

const (
	UnknownMsgType MessageType = iota
	AckMsgType
	MultipartMsgType
	HeartbeatMsgType
	ConnectedMsgType
	DisconnectedMsgType
	CreateAcctMsgType
	CreateAcctRespMsgType
	LoginMsgType
	LoginRespMsgType
	JoinGameMsgType
	GameConnectedMsgType
	GameMasterFrameMsgType
	EntityMsgType
	SnakeMsgType
	SetDirectionMsgType
	Vect2MsgType
)

// ParseNetMessage accepts input of raw bytes from a NetMessage. Parses and returns a Net message.
func ParseNetMessage(packet Packet, content []byte) Net {
	var msg Net
	switch packet.Frame.MsgType {
	case MultipartMsgType:
		msg = &Multipart{}
	case HeartbeatMsgType:
		msg = &Heartbeat{}
	case ConnectedMsgType:
		msg = &Connected{}
	case DisconnectedMsgType:
		msg = &Disconnected{}
	case CreateAcctMsgType:
		msg = &CreateAcct{}
	case CreateAcctRespMsgType:
		msg = &CreateAcctResp{}
	case LoginMsgType:
		msg = &Login{}
	case LoginRespMsgType:
		msg = &LoginResp{}
	case JoinGameMsgType:
		msg = &JoinGame{}
	case GameConnectedMsgType:
		msg = &GameConnected{}
	case GameMasterFrameMsgType:
		msg = &GameMasterFrame{}
	case EntityMsgType:
		msg = &Entity{}
	case SnakeMsgType:
		msg = &Snake{}
	case SetDirectionMsgType:
		msg = &SetDirection{}
	case Vect2MsgType:
		msg = &Vect2{}
	default:
		log.Printf("Unknown message type: %d", packet.Frame.MsgType)
		return nil
	}
	msg.Deserialize(bytes.NewBuffer(content))
	return msg
}

type Multipart struct {
	ID uint16
	GroupID uint32
	NumParts uint16
	Content []byte
}

func (m *Multipart) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	binary.Write(buffer, binary.LittleEndian, m.GroupID)
	binary.Write(buffer, binary.LittleEndian, m.NumParts)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Content)))
	buffer.Write(m.Content)
}

func (m *Multipart) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	binary.Read(buffer, binary.LittleEndian, &m.GroupID)
	binary.Read(buffer, binary.LittleEndian, &m.NumParts)
	var l3_1 int32
	binary.Read(buffer, binary.LittleEndian, &l3_1)
	m.Content = make([]byte, l3_1)
	for i := 0; i < int(l3_1); i++ {
		m.Content[i], _ = buffer.ReadByte()
	}
}

func (m *Multipart) Len() int {
	mylen := 0
	mylen += 2
	mylen += 4
	mylen += 2
	mylen += 4 + len(m.Content)
	return mylen
}

type Heartbeat struct {
	Time int64
	Latency int64
}

func (m *Heartbeat) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.Time)
	binary.Write(buffer, binary.LittleEndian, m.Latency)
}

func (m *Heartbeat) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.Time)
	binary.Read(buffer, binary.LittleEndian, &m.Latency)
}

func (m *Heartbeat) Len() int {
	mylen := 0
	mylen += 8
	mylen += 8
	return mylen
}

type Connected struct {
}

func (m *Connected) Serialize(buffer *bytes.Buffer) {
}

func (m *Connected) Deserialize(buffer *bytes.Buffer) {
}

func (m *Connected) Len() int {
	mylen := 0
	return mylen
}

type Disconnected struct {
}

func (m *Disconnected) Serialize(buffer *bytes.Buffer) {
}

func (m *Disconnected) Deserialize(buffer *bytes.Buffer) {
}

func (m *Disconnected) Len() int {
	mylen := 0
	return mylen
}

type CreateAcct struct {
	Name string
	Password string
}

func (m *CreateAcct) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Name)))
	buffer.WriteString(m.Name)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Password)))
	buffer.WriteString(m.Password)
}

func (m *CreateAcct) Deserialize(buffer *bytes.Buffer) {
	var l0_1 int32
	binary.Read(buffer, binary.LittleEndian, &l0_1)
	temp0_1 := make([]byte, l0_1)
	buffer.Read(temp0_1)
	m.Name = string(temp0_1)
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	temp1_1 := make([]byte, l1_1)
	buffer.Read(temp1_1)
	m.Password = string(temp1_1)
}

func (m *CreateAcct) Len() int {
	mylen := 0
	mylen += 4 + len(m.Name)
	mylen += 4 + len(m.Password)
	return mylen
}

type CreateAcctResp struct {
	AccountID uint32
	Name string
}

func (m *CreateAcctResp) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.AccountID)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Name)))
	buffer.WriteString(m.Name)
}

func (m *CreateAcctResp) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.AccountID)
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	temp1_1 := make([]byte, l1_1)
	buffer.Read(temp1_1)
	m.Name = string(temp1_1)
}

func (m *CreateAcctResp) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4 + len(m.Name)
	return mylen
}

type Login struct {
	Name string
	Password string
}

func (m *Login) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Name)))
	buffer.WriteString(m.Name)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Password)))
	buffer.WriteString(m.Password)
}

func (m *Login) Deserialize(buffer *bytes.Buffer) {
	var l0_1 int32
	binary.Read(buffer, binary.LittleEndian, &l0_1)
	temp0_1 := make([]byte, l0_1)
	buffer.Read(temp0_1)
	m.Name = string(temp0_1)
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	temp1_1 := make([]byte, l1_1)
	buffer.Read(temp1_1)
	m.Password = string(temp1_1)
}

func (m *Login) Len() int {
	mylen := 0
	mylen += 4 + len(m.Name)
	mylen += 4 + len(m.Password)
	return mylen
}

type LoginResp struct {
	Success byte
	Name string
	AccountID uint32
}

func (m *LoginResp) Serialize(buffer *bytes.Buffer) {
	buffer.WriteByte(m.Success)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Name)))
	buffer.WriteString(m.Name)
	binary.Write(buffer, binary.LittleEndian, m.AccountID)
}

func (m *LoginResp) Deserialize(buffer *bytes.Buffer) {
	m.Success, _ = buffer.ReadByte()
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	temp1_1 := make([]byte, l1_1)
	buffer.Read(temp1_1)
	m.Name = string(temp1_1)
	binary.Read(buffer, binary.LittleEndian, &m.AccountID)
}

func (m *LoginResp) Len() int {
	mylen := 0
	mylen += 1
	mylen += 4 + len(m.Name)
	mylen += 4
	return mylen
}

type JoinGame struct {
}

func (m *JoinGame) Serialize(buffer *bytes.Buffer) {
}

func (m *JoinGame) Deserialize(buffer *bytes.Buffer) {
}

func (m *JoinGame) Len() int {
	mylen := 0
	return mylen
}

type GameConnected struct {
	ID uint32
	TickID uint32
	Entities []*Entity
	Snakes []*Snake
}

func (m *GameConnected) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	binary.Write(buffer, binary.LittleEndian, m.TickID)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Entities)))
	for _, v2 := range m.Entities {
		v2.Serialize(buffer)
	}
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Snakes)))
	for _, v2 := range m.Snakes {
		v2.Serialize(buffer)
	}
}

func (m *GameConnected) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	binary.Read(buffer, binary.LittleEndian, &m.TickID)
	var l2_1 int32
	binary.Read(buffer, binary.LittleEndian, &l2_1)
	m.Entities = make([]*Entity, l2_1)
	for i := 0; i < int(l2_1); i++ {
		m.Entities[i] = new(Entity)
		m.Entities[i].Deserialize(buffer)
	}
	var l3_1 int32
	binary.Read(buffer, binary.LittleEndian, &l3_1)
	m.Snakes = make([]*Snake, l3_1)
	for i := 0; i < int(l3_1); i++ {
		m.Snakes[i] = new(Snake)
		m.Snakes[i].Deserialize(buffer)
	}
}

func (m *GameConnected) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4
	mylen += 4
	for _, v2 := range m.Entities {
	_ = v2
		mylen += v2.Len()
	}

	mylen += 4
	for _, v2 := range m.Snakes {
	_ = v2
		mylen += v2.Len()
	}

	return mylen
}

type GameMasterFrame struct {
	ID uint32
	Entities []*Entity
	Snakes []*Snake
	Tick uint32
}

func (m *GameMasterFrame) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Entities)))
	for _, v2 := range m.Entities {
		v2.Serialize(buffer)
	}
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Snakes)))
	for _, v2 := range m.Snakes {
		v2.Serialize(buffer)
	}
	binary.Write(buffer, binary.LittleEndian, m.Tick)
}

func (m *GameMasterFrame) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	m.Entities = make([]*Entity, l1_1)
	for i := 0; i < int(l1_1); i++ {
		m.Entities[i] = new(Entity)
		m.Entities[i].Deserialize(buffer)
	}
	var l2_1 int32
	binary.Read(buffer, binary.LittleEndian, &l2_1)
	m.Snakes = make([]*Snake, l2_1)
	for i := 0; i < int(l2_1); i++ {
		m.Snakes[i] = new(Snake)
		m.Snakes[i].Deserialize(buffer)
	}
	binary.Read(buffer, binary.LittleEndian, &m.Tick)
}

func (m *GameMasterFrame) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4
	for _, v2 := range m.Entities {
	_ = v2
		mylen += v2.Len()
	}

	mylen += 4
	for _, v2 := range m.Snakes {
	_ = v2
		mylen += v2.Len()
	}

	mylen += 4
	return mylen
}

type Entity struct {
	ID uint32
	EType uint16
	ContainerID uint32
	X int32
	Y int32
	Size int32
	Facing *Vect2
}

func (m *Entity) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	binary.Write(buffer, binary.LittleEndian, m.EType)
	binary.Write(buffer, binary.LittleEndian, m.ContainerID)
	binary.Write(buffer, binary.LittleEndian, m.X)
	binary.Write(buffer, binary.LittleEndian, m.Y)
	binary.Write(buffer, binary.LittleEndian, m.Size)
	m.Facing.Serialize(buffer)
}

func (m *Entity) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	binary.Read(buffer, binary.LittleEndian, &m.EType)
	binary.Read(buffer, binary.LittleEndian, &m.ContainerID)
	binary.Read(buffer, binary.LittleEndian, &m.X)
	binary.Read(buffer, binary.LittleEndian, &m.Y)
	binary.Read(buffer, binary.LittleEndian, &m.Size)
	m.Facing = new(Vect2)
	m.Facing.Deserialize(buffer)
}

func (m *Entity) Len() int {
	mylen := 0
	mylen += 4
	mylen += 2
	mylen += 4
	mylen += 4
	mylen += 4
	mylen += 4
	mylen += m.Facing.Len()
	return mylen
}

type Snake struct {
	ID uint32
	Segments []uint32
	Speed int32
}

func (m *Snake) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	binary.Write(buffer, binary.LittleEndian, int32(len(m.Segments)))
	for _, v2 := range m.Segments {
		binary.Write(buffer, binary.LittleEndian, v2)
	}
	binary.Write(buffer, binary.LittleEndian, m.Speed)
}

func (m *Snake) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	var l1_1 int32
	binary.Read(buffer, binary.LittleEndian, &l1_1)
	m.Segments = make([]uint32, l1_1)
	for i := 0; i < int(l1_1); i++ {
		binary.Read(buffer, binary.LittleEndian, &m.Segments[i])
	}
	binary.Read(buffer, binary.LittleEndian, &m.Speed)
}

func (m *Snake) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4
	for _, v2 := range m.Segments {
	_ = v2
		mylen += 4
	}

	mylen += 4
	return mylen
}

type SetDirection struct {
	ID uint32
	Facing *Vect2
	TickID uint32
}

func (m *SetDirection) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.ID)
	m.Facing.Serialize(buffer)
	binary.Write(buffer, binary.LittleEndian, m.TickID)
}

func (m *SetDirection) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.ID)
	m.Facing = new(Vect2)
	m.Facing.Deserialize(buffer)
	binary.Read(buffer, binary.LittleEndian, &m.TickID)
}

func (m *SetDirection) Len() int {
	mylen := 0
	mylen += 4
	mylen += m.Facing.Len()
	mylen += 4
	return mylen
}

type Vect2 struct {
	X int32
	Y int32
}

func (m *Vect2) Serialize(buffer *bytes.Buffer) {
	binary.Write(buffer, binary.LittleEndian, m.X)
	binary.Write(buffer, binary.LittleEndian, m.Y)
}

func (m *Vect2) Deserialize(buffer *bytes.Buffer) {
	binary.Read(buffer, binary.LittleEndian, &m.X)
	binary.Read(buffer, binary.LittleEndian, &m.Y)
}

func (m *Vect2) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4
	return mylen
}

