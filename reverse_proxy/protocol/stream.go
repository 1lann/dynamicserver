package protocol

import (
	"io"
	"log"
)

type Stream struct {
	io.ReadWriter
}

type PacketStream struct {
	Stream
	reader *io.LimitedReader
}

type readWriter struct {
	io.Reader
	io.Writer
}

func (s PacketStream) ExhaustPacket() error {
	if s.reader.N == 0 {
		return nil
	}

	log.Println("[WARNING]", s.reader.N, " bytes of data exhausted!")
	return s.ReadFull(make([]byte, s.reader.N))
}

func (s Stream) GetPacketStream() (PacketStream, int, error) {
	length, err := s.ReadVarInt()
	if err != nil {
		return PacketStream{}, 0, err
	}

	if length == 0 {
		return PacketStream{}, 0, ErrInvalidData
	}

	limitedReader := &io.LimitedReader{R: s, N: int64(length)}

	return PacketStream{Stream{readWriter{limitedReader, s}}, limitedReader},
		length, nil
}

func NewStream(readWriter io.ReadWriter) Stream {
	return Stream{readWriter}
}

// Returns decoded (little endian) data of length data.
func (s Stream) DecodeReadFull(data []byte) error {
	_, err := io.ReadFull(s, data)
	if err != nil {
		return err
	}

	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return nil
}

func (s Stream) ReadFull(data []byte) error {
	_, err := io.ReadFull(s, data)
	if err != nil {
		return err
	}

	return nil
}

func (s Stream) WritePacket(p *Packet) error {

	writePacket := &Packet{}
	writePacket.WriteVarInt(len(p.Data))
	writePacket.Data = append(writePacket.Data, p.Data...)

	_, err := s.Write(writePacket.Data)
	if err != nil {
		return err
	}

	return nil
}
