package main

import (
	"github.com/1lann/dynamicserver/reverse_proxy/ping"
	"github.com/1lann/dynamicserver/reverse_proxy/protocol"
	"io"
	"log"
	"net"
	"strings"
)

func main() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		log.Println("Failed to listen", err)
	}

	log.Println("Listening...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept", err)
			return
		}

		go handleConnection(conn)
	}
}

func handleLoginStage(stream protocol.Stream, ipAddr string) {
	for {
		packetStream, _, err := stream.GetPacketStream()
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Println(err)
			return
		}

		// wrappedStream := protocol.Stream{packetStream}

		packetId, err := packetStream.ReadVarInt()
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("[LOGIN] Received packet with ID:", packetId)

		if packetId == 0 {
			username, err := packetStream.ReadString()
			if err != nil {
				log.Println("[LOGIN] Failed to read username:", err)
				return
			}

			log.Println("[LOGIN] Connection request:", username)

			err = ping.RejectWithMessage(stream,
				"Hey there! This is a test server made by 1lann.\n"+
					"I can see you're logging in as "+username+"\n"+
					"and your IP address is "+ipAddr)
			if err != nil {
				log.Println("[LOGIN] Error while rejecting:", err)
				return
			}
		}
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	ipAddr := strings.Split(conn.RemoteAddr().String(), ":")[0]

	stream := protocol.NewStream(conn)

	hasHandshake := false

	for {
		packetStream, length, err := stream.GetPacketStream()
		if err != nil {
			if err == io.EOF {
				return
			}

			log.Println(err)
			return
		}

		wrappedStream := protocol.Stream{packetStream}

		packetId, err := packetStream.ReadVarInt()
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Received packet with ID:", packetId)
		if packetId == 0 && length > 4 {
			pingPacket, err := ping.ReadHandshakePacket(wrappedStream)
			if err != nil {
				log.Println("Ping packet read error:", err)
				log.Println(pingPacket)
				return
			}

			if pingPacket.NextState == 1 && !hasHandshake {
				log.Println("Receive ping packet:", pingPacket)
				hasHandshake = true

				err = ping.WriteHandshakeResponse(wrappedStream, 420,
					"Your IP is: "+ipAddr)
				if err != nil {
					log.Println(err)
				}
			} else if pingPacket.NextState == 2 {
				// Start login process
				packetStream.ExhaustPacket()
				handleLoginStage(stream, ipAddr)
			}
		} else if packetId == 1 {
			err := ping.HandlePingPacket(wrappedStream)
			if err != nil {
				log.Println("Pinging error:", err)
				return
			}
		}

		packetStream.ExhaustPacket()
	}
}
