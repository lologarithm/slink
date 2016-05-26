package messages

import (
	"encoding/binary"
	"log"
	"math"
)

type Net interface {
	Serialize([]byte)
	Deserialize([]byte)
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
	TurnSnakeMsgType
	RemoveEntityMsgType
	UpdateEntityMsgType
	SnakeDiedMsgType
	Vect2MsgType
	AMsgType
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
	case TurnSnakeMsgType:
		msg = &TurnSnake{}
	case RemoveEntityMsgType:
		msg = &RemoveEntity{}
	case UpdateEntityMsgType:
		msg = &UpdateEntity{}
	case SnakeDiedMsgType:
		msg = &SnakeDied{}
	case Vect2MsgType:
		msg = &Vect2{}
	case AMsgType:
		msg = &A{}
	default:
		log.Printf("Unknown message type: %d", packet.Frame.MsgType)
		return nil
	}
	msg.Deserialize(content)
	return msg
}

type Multipart struct {
	ID uint16
	GroupID uint32
	NumParts uint16
	Content []byte
}

func (m *Multipart) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint16(buffer[idx:], uint16(m.ID))
	idx+=2
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.GroupID))
	idx+=4
	binary.LittleEndian.PutUint16(buffer[idx:], uint16(m.NumParts))
	idx+=2
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Content)))
	idx += 4
	copy(buffer[idx:], m.Content)
	idx+=len(m.Content)

	_ = idx
}

func (m *Multipart) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint16(buffer[idx:])
	idx+=2
	m.GroupID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	m.NumParts = binary.LittleEndian.Uint16(buffer[idx:])
	idx+=2
	l3_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Content = make([]byte, l3_1)
	for i := 0; i < int(l3_1); i++ {
		m.Content[i] = buffer[idx]

		idx+=1
	}

	_ = idx
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

func (m *Heartbeat) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint64(buffer[idx:], uint64(m.Time))
	idx+=8
	binary.LittleEndian.PutUint64(buffer[idx:], uint64(m.Latency))
	idx+=8

	_ = idx
}

func (m *Heartbeat) Deserialize(buffer []byte) {
	idx := 0
	m.Time = int64(binary.LittleEndian.Uint64(buffer[idx:]))
	idx+=8
	m.Latency = int64(binary.LittleEndian.Uint64(buffer[idx:]))
	idx+=8

	_ = idx
}

func (m *Heartbeat) Len() int {
	mylen := 0
	mylen += 8
	mylen += 8
	return mylen
}

type Connected struct {
}

func (m *Connected) Serialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *Connected) Deserialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *Connected) Len() int {
	mylen := 0
	return mylen
}

type Disconnected struct {
}

func (m *Disconnected) Serialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *Disconnected) Deserialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *Disconnected) Len() int {
	mylen := 0
	return mylen
}

type CreateAcct struct {
	Name string
	Password string
}

func (m *CreateAcct) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Password)))
	idx += 4
	copy(buffer[idx:], []byte(m.Password))
	idx+=len(m.Password)

	_ = idx
}

func (m *CreateAcct) Deserialize(buffer []byte) {
	idx := 0
	l0_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l0_1])
	idx+=len(m.Name)
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Password = string(buffer[idx:idx+l1_1])
	idx+=len(m.Password)

	_ = idx
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

func (m *CreateAcctResp) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.AccountID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)

	_ = idx
}

func (m *CreateAcctResp) Deserialize(buffer []byte) {
	idx := 0
	m.AccountID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l1_1])
	idx+=len(m.Name)

	_ = idx
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

func (m *Login) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Password)))
	idx += 4
	copy(buffer[idx:], []byte(m.Password))
	idx+=len(m.Password)

	_ = idx
}

func (m *Login) Deserialize(buffer []byte) {
	idx := 0
	l0_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l0_1])
	idx+=len(m.Name)
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Password = string(buffer[idx:idx+l1_1])
	idx+=len(m.Password)

	_ = idx
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

func (m *LoginResp) Serialize(buffer []byte) {
	idx := 0
	buffer[idx] = m.Success
	idx+=1
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.AccountID))
	idx+=4

	_ = idx
}

func (m *LoginResp) Deserialize(buffer []byte) {
	idx := 0
	m.Success = buffer[idx]

	idx+=1
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l1_1])
	idx+=len(m.Name)
	m.AccountID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4

	_ = idx
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

func (m *JoinGame) Serialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *JoinGame) Deserialize(buffer []byte) {
	idx := 0

	_ = idx
}

func (m *JoinGame) Len() int {
	mylen := 0
	return mylen
}

type GameConnected struct {
	ID uint32
	SnakeID uint32
	TickID uint32
	Entities []*Entity
	Snakes []*Snake
}

func (m *GameConnected) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.SnakeID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.TickID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Entities)))
	idx += 4
	for _, v2 := range m.Entities {
		v2.Serialize(buffer[idx:])
		idx+=v2.Len()
	}
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Snakes)))
	idx += 4
	for _, v2 := range m.Snakes {
		v2.Serialize(buffer[idx:])
		idx+=v2.Len()
	}

	_ = idx
}

func (m *GameConnected) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	m.SnakeID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	m.TickID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	l3_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Entities = make([]*Entity, l3_1)
	for i := 0; i < int(l3_1); i++ {
		m.Entities[i] = new(Entity)
		m.Entities[i].Deserialize(buffer[idx:])
idx+=m.Entities[i].Len()
	}
	l4_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Snakes = make([]*Snake, l4_1)
	for i := 0; i < int(l4_1); i++ {
		m.Snakes[i] = new(Snake)
		m.Snakes[i].Deserialize(buffer[idx:])
idx+=m.Snakes[i].Len()
	}

	_ = idx
}

func (m *GameConnected) Len() int {
	mylen := 0
	mylen += 4
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

func (m *GameMasterFrame) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Entities)))
	idx += 4
	for _, v2 := range m.Entities {
		v2.Serialize(buffer[idx:])
		idx+=v2.Len()
	}
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Snakes)))
	idx += 4
	for _, v2 := range m.Snakes {
		v2.Serialize(buffer[idx:])
		idx+=v2.Len()
	}
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Tick))
	idx+=4

	_ = idx
}

func (m *GameMasterFrame) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Entities = make([]*Entity, l1_1)
	for i := 0; i < int(l1_1); i++ {
		m.Entities[i] = new(Entity)
		m.Entities[i].Deserialize(buffer[idx:])
idx+=m.Entities[i].Len()
	}
	l2_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Snakes = make([]*Snake, l2_1)
	for i := 0; i < int(l2_1); i++ {
		m.Snakes[i] = new(Snake)
		m.Snakes[i].Deserialize(buffer[idx:])
idx+=m.Snakes[i].Len()
	}
	m.Tick = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4

	_ = idx
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
	X int32
	Y int32
	Size int32
	Facing *Vect2
}

func (m *Entity) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4
	binary.LittleEndian.PutUint16(buffer[idx:], uint16(m.EType))
	idx+=2
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.X))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Y))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Size))
	idx+=4
	m.Facing.Serialize(buffer[idx:])
	idx+=m.Facing.Len()

	_ = idx
}

func (m *Entity) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	m.EType = binary.LittleEndian.Uint16(buffer[idx:])
	idx+=2
	m.X = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Y = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Size = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Facing = new(Vect2)
	m.Facing.Deserialize(buffer[idx:])
idx+=m.Facing.Len()

	_ = idx
}

func (m *Entity) Len() int {
	mylen := 0
	mylen += 4
	mylen += 2
	mylen += 4
	mylen += 4
	mylen += 4
	mylen += m.Facing.Len()
	return mylen
}

type Snake struct {
	ID uint32
	Name string
	Segments []uint32
	Speed int32
	Turning int16
}

func (m *Snake) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Segments)))
	idx += 4
	for _, v2 := range m.Segments {
		binary.LittleEndian.PutUint32(buffer[idx:], uint32(v2))
		idx+=4
	}
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Speed))
	idx+=4
	binary.LittleEndian.PutUint16(buffer[idx:], uint16(m.Turning))
	idx+=2

	_ = idx
}

func (m *Snake) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	l1_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l1_1])
	idx+=len(m.Name)
	l2_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Segments = make([]uint32, l2_1)
	for i := 0; i < int(l2_1); i++ {
		m.Segments[i] = binary.LittleEndian.Uint32(buffer[idx:])
		idx+=4
	}
	m.Speed = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Turning = int16(binary.LittleEndian.Uint16(buffer[idx:]))
	idx+=2

	_ = idx
}

func (m *Snake) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4 + len(m.Name)
	mylen += 4
	for _, v2 := range m.Segments {
	_ = v2
		mylen += 4
	}

	mylen += 4
	mylen += 2
	return mylen
}

type TurnSnake struct {
	ID uint32
	Direction int16
	TickID uint32
}

func (m *TurnSnake) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4
	binary.LittleEndian.PutUint16(buffer[idx:], uint16(m.Direction))
	idx+=2
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.TickID))
	idx+=4

	_ = idx
}

func (m *TurnSnake) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4
	m.Direction = int16(binary.LittleEndian.Uint16(buffer[idx:]))
	idx+=2
	m.TickID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4

	_ = idx
}

func (m *TurnSnake) Len() int {
	mylen := 0
	mylen += 4
	mylen += 2
	mylen += 4
	return mylen
}

type RemoveEntity struct {
	Ent *Entity
}

func (m *RemoveEntity) Serialize(buffer []byte) {
	idx := 0
	m.Ent.Serialize(buffer[idx:])
	idx+=m.Ent.Len()

	_ = idx
}

func (m *RemoveEntity) Deserialize(buffer []byte) {
	idx := 0
	m.Ent = new(Entity)
	m.Ent.Deserialize(buffer[idx:])
idx+=m.Ent.Len()

	_ = idx
}

func (m *RemoveEntity) Len() int {
	mylen := 0
	mylen += m.Ent.Len()
	return mylen
}

type UpdateEntity struct {
	Ent *Entity
}

func (m *UpdateEntity) Serialize(buffer []byte) {
	idx := 0
	m.Ent.Serialize(buffer[idx:])
	idx+=m.Ent.Len()

	_ = idx
}

func (m *UpdateEntity) Deserialize(buffer []byte) {
	idx := 0
	m.Ent = new(Entity)
	m.Ent.Deserialize(buffer[idx:])
idx+=m.Ent.Len()

	_ = idx
}

func (m *UpdateEntity) Len() int {
	mylen := 0
	mylen += m.Ent.Len()
	return mylen
}

type SnakeDied struct {
	ID uint32
}

func (m *SnakeDied) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.ID))
	idx+=4

	_ = idx
}

func (m *SnakeDied) Deserialize(buffer []byte) {
	idx := 0
	m.ID = binary.LittleEndian.Uint32(buffer[idx:])
	idx+=4

	_ = idx
}

func (m *SnakeDied) Len() int {
	mylen := 0
	mylen += 4
	return mylen
}

type Vect2 struct {
	X int32
	Y int32
}

func (m *Vect2) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.X))
	idx+=4
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Y))
	idx+=4

	_ = idx
}

func (m *Vect2) Deserialize(buffer []byte) {
	idx := 0
	m.X = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Y = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4

	_ = idx
}

func (m *Vect2) Len() int {
	mylen := 0
	mylen += 4
	mylen += 4
	return mylen
}

type A struct {
	Name string
	BirthDay int64
	Phone string
	Siblings int32
	Spouse byte
	Money float64
}

func (m *A) Serialize(buffer []byte) {
	idx := 0
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Name)))
	idx += 4
	copy(buffer[idx:], []byte(m.Name))
	idx+=len(m.Name)
	binary.LittleEndian.PutUint64(buffer[idx:], uint64(m.BirthDay))
	idx+=8
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(len(m.Phone)))
	idx += 4
	copy(buffer[idx:], []byte(m.Phone))
	idx+=len(m.Phone)
	binary.LittleEndian.PutUint32(buffer[idx:], uint32(m.Siblings))
	idx+=4
	buffer[idx] = m.Spouse
	idx+=1
	binary.LittleEndian.PutUint64(buffer[idx:], math.Float64bits(m.Money))
	idx+=8

	_ = idx
}

func (m *A) Deserialize(buffer []byte) {
	idx := 0
	l0_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Name = string(buffer[idx:idx+l0_1])
	idx+=len(m.Name)
	m.BirthDay = int64(binary.LittleEndian.Uint64(buffer[idx:]))
	idx+=8
	l2_1 := int(binary.LittleEndian.Uint32(buffer[idx:]))
	idx += 4
	m.Phone = string(buffer[idx:idx+l2_1])
	idx+=len(m.Phone)
	m.Siblings = int32(binary.LittleEndian.Uint32(buffer[idx:]))
	idx+=4
	m.Spouse = buffer[idx]

	idx+=1
	m.Money = math.Float64frombits(binary.LittleEndian.Uint64(buffer[idx:]))
	idx+=8

	_ = idx
}

func (m *A) Len() int {
	mylen := 0
	mylen += 4 + len(m.Name)
	mylen += 8
	mylen += 4 + len(m.Phone)
	mylen += 4
	mylen += 1
	mylen += 8
	return mylen
}

