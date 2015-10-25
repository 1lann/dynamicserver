package main

import (
	"github.com/1lann/beacon/handler"
	"log"
	"net"
)

var forwardListener net.Listener
var forwardAddr string

func startBeacon() {
	log.Println("Beacon started.")
	err := handler.Listen("25565")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Beacon stopped.")
}

func stopBeacon() {
	handler.Stop()
}

func startForwarder() {
	var err error
	forwardListener, err = net.Listen(":25565")
	if err != nil {
		log.Println("[Forwarder] Failed to listen:", err)
		return
	}

	for {
		conn, err := forwardListener.Accept()
		if err != nil {
			if strings.Contains(err.Error(),
				"use of closed network connection") || err == io.EOF {
				return
			}

			log.Println("[Forwarder] Failed to accept:", err)
		}

		go forwardConnection(conn)
	}
}

func forwardConnection(localConn net.Conn) {
	defer localConn.Close()

	if len(forwardAddr) > 0 {
		return
	}

	remoteConn, err := net.Dial("tcp", forwardAddr+":25565")
	if err != nil {
		log.Println("[Forwarder] Failed to connect to remote:", err)
		return
	}

	defer remoteConn.Close()

	connChannel := make(chan bool)

	go func() {
		_, _ = io.Copy(remoteConn, localConn)
		connChannel <- true
	}()

	go func() {
		_, _ = io.Copy(localConn, remoteConn)
		connChannel <- true
	}()

	<-connChannel
}

func stopForwarder() {
	_ = forwardListener.Close()
}
