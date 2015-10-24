package network

import (
	"bytes"
	"encoding/binary"
)

var ErrInvalidData = errors.New("network: invalid data.")

func (s Stream) ReadByte() (byte, error) {
	data := make([]byte, 1)
	numRead, err := s.Read(p)
	if err != nil {
		return 0, err
	}

	return p[0], nil
}

func (s Stream) ReadBoolean() (bool, error) {
	b, err := s.ReadByte()
	if err != nil {
		return false, err
	}

	if b == 0x01 {
		return true
	} else if b == 0x00 {
		return false
	} else {
		return false, ErrInvalidData
	}
}

func (s Stream) ReadSignedByte() (int8, error) {
	b, err := s.ReadByte()
	if err != nil {
		return false, err
	}

	return int8(b), nil
}

// Also known as ReadShort
func (s Stream) ReadInt16() (int16, error) {
	b := make([]byte, 2)
	err := s.DecodeReadFull(b)
	if err != nil {
		return 0, err
	}

	return int16(binary.LittleEndian.UInt16(b)), nil
}

// Also known as ReadUnsignedShort
func (s Stream) ReadUInt16() (uint16, error) {
	b := make([]byte, 2)
	err := s.DecodeReadFull(b)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint16(b), nil
}

func (s Stream) ReadInt32() (int32, error) {
	b := make([]byte, 4)
	err := s.DecodeReadFull(b)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Int32(b), nil
}

func (s Stream) ReadInt64() (int64, error) {
	b := make([]byte, 8)
	err := s.DecodeReadFull(b)
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Int64(b), nil
}

func (s Stream) ReadFloat32() (float32, error) {
	var result float32
	err := binary.Read(s, binary.LittleEndian, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (s Stream) ReadFloat64() (float64, error) {
	var result float64
	err := binary.Read(s, binary.LittleEndian, &result)
	if err != nil {
		return 0, err
	}

	return result, nil
}

func (s Stream) ReadString() (string, error) {
	length, err := binary.ReadVarint(s)
	if err != nil {
		return "", err
	}

	data := make([]byte, length)
	err = s.DecodeReadFull(data)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s Stream) ReadVarInt() (int, error) {
	num, err := binary.ReadVarint(s)
	if err != nil {
		return 0, err
	}

	return int(num)
}

func (s Stream) ReadVarLong() (int64, error) {
	num, err := binary.ReadVarint(s)
	if err != nil {
		return 0, err
	}

	return int64(num)
}
