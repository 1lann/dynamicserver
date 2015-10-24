package network

import (
	"bytes"
	"encoding/binary"
)

// Packet stores data as BigEndian
type Packet struct {
	Data []byte
}

func NewPacketWithId(id int) *Packet {
	p := &Packet{}
	p.WriteVarInt(id)
	return p
}

func (p *Packet) Write(data []byte) (int, error) {
	p.Data = append(p.Data, data...)
	return len(data), nil
}

func (p *Packet) WriteByte(data byte) {
	p.Data = append(p.Data, data)
}

func (p *Packet) WriteBoolean(data bool) {
	if data {
		p.Data = append(p.Data, 0x01)
	} else {
		p.Data = append(p.Data, 0x00)
	}
}

func (p *Packet) WriteSignedByte(data int8) {
	p.Data = append(p.Data, byte(data))
}

func (p *Packet) WriteInt16(data int16) {
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, uint16(data))
	p.Data = append(p.Data, result...)
}

func (p *Packet) WriteUInt16(data uint16) {
	result := make([]byte, 2)
	binary.BigEndian.PutUint16(result, data)
	p.Data = append(p.Data, result...)
}

func (p *Packet) WriteInt32(data int32) {
	result := make([]byte, 4)
	binary.BigEndian.PutUint32(result, uint32(data))
	p.Data = append(p.Data, result...)
}

func (p *Packet) WriteInt64(data int64) {
	result := make([]byte, 8)
	binary.BigEndian.PutUint64(result, uint64(data))
	p.Data = append(p.Data, result...)
}

func (p *Packet) WriteFloat32(data float32) {
	_ = binary.Write(p, binary.BigEndian, data)
	return
}

func (p *Packet) WriteFloat64(data float64) {
	_ = binary.Write(p, binary.BigEndian, data)
	return
}

func (p *Packet) WriteString(data string) {
	_ = binary.PutVarint(p, int64(len(data)))
	p.Data = append(p.Data, []byte(data)...)
}

func (p *Packet) WriteVarInt(data int) {
	result := make([]byte, 10)
	numBytes := binary.PutVarint(result, int64(data))
	p.Data = append(p.Data, result[:numBytes]...)
}

func (p *Packet) WriteVarLong(data int64) {
	result := make([]byte, 10)
	numBytes := binary.PutVarint(result, data)
	p.Data = append(p.Data, result[:numBytes]...)
}
