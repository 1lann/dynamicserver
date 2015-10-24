package ping

import (
	"encoding/json"
	"github.com/1lann/dynamicserver/reverse_proxy/protocol"
)

type HandshakePacket struct {
	ProtocolVersion int
	ServerAddress   string
	ServerPort      uint16
	NextState       int
}

type ChatMessage struct {
	Text string `json:"text"`
}

type StatusResponse struct {
	Version     `json:"version"`
	Players     `json:"players"`
	Description ChatMessage `json:"description"`
}

type Version struct {
	Name     string `json:"name"`
	Protocol int    `json:"protocol"`
}

type Players struct {
	Max    int `json:"max"`
	Online int `json:"online"`
}

func ReadHandshakePacket(s protocol.Stream) (HandshakePacket, error) {
	handshake := HandshakePacket{}
	var err error
	if handshake.ProtocolVersion, err = s.ReadVarInt(); err != nil {
		return HandshakePacket{}, err
	}

	if handshake.ServerAddress, err = s.ReadString(); err != nil {
		return HandshakePacket{}, err
	}

	if handshake.ServerPort, err = s.ReadUInt16(); err != nil {
		return HandshakePacket{}, err
	}

	handshake.NextState, err = s.ReadVarInt()
	return handshake, err
}

func WriteHandshakeResponse(s protocol.Stream, maxPlayers int,
	statusMessage string) error {
	statusResponse := StatusResponse{
		Version: Version{
			Name:     "1.7.10",
			Protocol: 5,
		},
		Players: Players{
			Max:    maxPlayers,
			Online: 0,
		},
		Description: ChatMessage{
			Text: statusMessage,
		},
	}

	data, err := json.Marshal(statusResponse)
	if err != nil {
		return err
	}

	responsePacket := protocol.NewPacketWithId(0x00)
	responsePacket.WriteString(string(data))
	err = s.WritePacket(responsePacket)
	return err
}

func HandlePingPacket(s protocol.Stream) error {
	time, err := s.ReadInt64()
	if err != nil {
		return err
	}
	responsePacket := protocol.NewPacketWithId(0x01)
	responsePacket.WriteInt64(time)
	err = s.WritePacket(responsePacket)
	return err
}
