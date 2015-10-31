package main

import (
	"github.com/1lann/beacon/protocol"
	"net"
	"time"
)

func (s *Server) IsMinecraftServerResponding() bool {
	conn, err := net.Dial("tcp", s.IPAddress+":25565")
	if err != nil {
		return false
	}

	defer conn.Close()

	stream := protocol.NewStream(conn)
	handshake := protocol.NewPacketWithId(0x00)
	handshake.WriteVarInt(5)
	handshake.WriteString(s.IPAddress)
	handshake.WriteUInt16(25565)
	handshake.WriteVarInt(1)
	if err := stream.WritePacket(handshake); err != nil {
		return false
	}

	request := protocol.NewPacketWithId(0x00)
	if err := stream.WritePacket(request); err != nil {
		return false
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	_, length, err := stream.GetPacketStream()
	if err != nil {
		return false
	}

	if length > 5 {
		return true
	}

	return false
}

func startResponseMonitor() {
	for {
		for _, server := range allServers {
			server.StateLock.Lock()

			if server.IsMinecraftServerResponding() {
				server.SetState(stateStarted)
			} else {
				if server.State == stateStarted {
					server.SetState(stateUnavailable)
				}
			}

			server.StateLock.Unlock()
		}
	}
}
