package messages

import (
	"encoding/binary"
	"fmt"
)

const FrameLen int = 6

func NewPacket(t MessageType, msg Net) *Packet {
	return &Packet{
		Frame: Frame{
			MsgType:       t,
			ContentLength: uint16(msg.Len()),
		},
		NetMsg: msg,
	}
}

type Packet struct {
	Frame  Frame
	NetMsg Net
}

// Pack serializes the content into RawBytes.
func (m *Packet) Pack() []byte {
	buf := make([]byte, m.Len())
	binary.LittleEndian.PutUint16(buf, uint16(m.Frame.MsgType))
	binary.LittleEndian.PutUint16(buf[2:], m.Frame.Seq)
	binary.LittleEndian.PutUint16(buf[4:], m.Frame.ContentLength)
	m.NetMsg.Serialize(buf[6:])
	return buf
}

// Len returns the total length of the message including the frame
func (m *Packet) Len() int {
	return int(m.Frame.ContentLength) + FrameLen
}

type Frame struct {
	MsgType       MessageType // byte 0-1, type
	Seq           uint16      // byte 2-3, order of message
	ContentLength uint16      // byte 4-5, content length
}

func (mf Frame) String() string {
	return fmt.Sprintf("Type: %d, Seq: %d, CL: %d\n", mf.MsgType, mf.Seq, mf.ContentLength)
}

func ParseFrame(rawBytes []byte) (mf Frame, ok bool) {
	if len(rawBytes) < FrameLen {
		return
	}
	mf.MsgType = MessageType(binary.LittleEndian.Uint16(rawBytes[0:2]))
	mf.Seq = binary.LittleEndian.Uint16(rawBytes[2:4])
	mf.ContentLength = binary.LittleEndian.Uint16(rawBytes[4:6])
	return mf, true
}

func NextPacket(rawBytes []byte) (packet Packet, ok bool) {
	packet.Frame, ok = ParseFrame(rawBytes)
	if !ok {
		return
	}

	ok = false
	if packet.Len() <= len(rawBytes) {
		packet.NetMsg = ParseNetMessage(packet, rawBytes[FrameLen:packet.Len()])
		if packet.NetMsg != nil {
			ok = true
		}
	}

	return
}
