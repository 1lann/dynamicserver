package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":25565")
	if err != nil {
		log.Println("Failed to listen", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept", err)
			return
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {

}
