package main

import (
	"github.com/1lann/beacon/protocol"
	"net"
	"time"
)

func (s *Server) IsMinecraftServerResponding() bool {
	conn, err := net.DialTimeout("tcp", s.IPAddress+":25565", time.Second*5)
	if err != nil {
		return false
	}

	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Second * 5))

	stream := protocol.NewStream(conn)
	handshake := protocol.NewPacketWithID(0x00)
	handshake.WriteVarInt(5)
	handshake.WriteString(s.IPAddress)
	handshake.WriteUInt16(25565)
	handshake.WriteVarInt(1)
	if err := stream.WritePacket(handshake); err != nil {
		return false
	}

	request := protocol.NewPacketWithID(0x00)
	if err := stream.WritePacket(request); err != nil {
		return false
	}

	conn.SetDeadline(time.Now().Add(time.Second * 5))

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
			if !server.Available {
				continue
			}

			server.StateLock.Lock()

			if server.IsMinecraftServerResponding() {
				if server.State != stateShutdown &&
					server.State != stateSnapshot &&
					server.State != stateDestroy {
					server.SetState(stateStarted)
				}
			} else {
				if server.State == stateStarted {
					server.SetState(stateUnavailable)
				}
			}

			server.StateLock.Unlock()
		}

		time.Sleep(time.Second * 5)
	}
}
