package network

import (
	"bytes"
	"encoding/binary"
	"io"
)

type Stream struct {
	io.ReadWriter
}

// Returns decoded (little endian) data of length data.
func (s Stream) DecodeReadFull(data []byte) error {
	_, err := ReadFull(s, data)
	if err != nil {
		return err
	}

	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}

func (s Stream) WritePacket(p *Packet) error {
	result := make([]byte, 10)
	numBytes := binary.PutVarint(result, int64(len(p.Data)))
	data := result[:numBytes]
	_, err := s.Write(data)
	if err != nil {
		return err
	}
	_, err = s.Write(p.Data)
	if err != nil {
		return err
	}

	return nil
}
