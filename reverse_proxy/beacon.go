package main

import (
	"github.com/1lann/beacon/handler"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

var forwardListener net.Listener
var forwardAddr string
var connectionLock *sync.Mutex = &sync.Mutex{}

func startBeacon() {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	log.Println("[Beacon] Started.")
	err := handler.Listen("25565")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[Beacon] Stopped.")
}

func stopBeacon() {
	handler.Stop()
}

func startForwarder() {
	connectionLock.Lock()
	defer connectionLock.Unlock()

	var err error
	forwardListener, err = net.Listen("tcp", ":25565")
	if err != nil {
		log.Println("[Forwarder] Failed to listen:", err)
		return
	}

	log.Println("[Forwarder] Started.")

	for {
		conn, err := forwardListener.Accept()
		if err != nil {
			if strings.Contains(err.Error(),
				"use of closed network connection") || err == io.EOF {
				log.Println("[Forwarder] Stopped.")
				return
			}

			log.Println("[Forwarder] Failed to accept:", err)
			return
		}

		go forwardConnection(conn)
	}
}

func forwardConnection(localConn net.Conn) {
	defer localConn.Close()

	if forwardAddr == "" {
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
