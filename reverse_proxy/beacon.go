package main

import (
	"github.com/1lann/beacon/chat"
	"github.com/1lann/beacon/handler"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

type state int

var currentState state

const (
	stateInitializing = iota
	stateStopped
	stateOff
	stateShutdown
	stateSnapshot
	stateDestroy
	stateStarted
	stateStarting
	stateUnavailable
)

const prefixText = chat.Aqua + "[chuie.io] " + chat.White
const headerText = chat.Aqua + "-- [ chuie.io ] --\n\n" + chat.White

func (s state) String() string {
	switch s {
	case stateInitializing:
		return "Intializing"
	case stateStopped:
		return "Stopped"
	case stateOff:
		return "Off"
	case stateSnapshot:
		return "Snapshot"
	case stateShutdown:
		return "Shutdown"
	case stateDestroy:
		return "Destroy"
	case stateStarted:
		return "Started"
	case stateStarting:
		return "Starting"
	}

	return "Unknown"
}

var connectMessage string

func setState(s state) {
	// if time.Now().Sub(lastSet) < time.Duration(time.Second*2) {
	// 	log.Println("Set state cancelled due to being too fast")
	// }

	// lastSet := time.Now()

	if s != stateStarted && currentState == stateStarted {
		stopForwarder()
		go startBeacon()
	}

	switch s {
	case stateInitializing:
		connectMessage = "Sorry, the server is not ready take requests yet!\n\n" +
			"Try connecting again in a few seconds."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Yellow + "Initializing..."
		handler.CurrentStatus.ShowConnection = false
	case stateStopped:
		connectMessage = "Sorry, the server is intentionally down.\n\n" +
			"Contact the server owner for more information."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Red + "Intentionally down"
		handler.CurrentStatus.ShowConnection = false
	case stateShutdown, stateSnapshot, stateDestroy:
		connectMessage = "Sorry, the server is currently shutting down.\n" +
			"You may start it again when it is completely turned off.\n\n" +
			"Try connecting again in a few minutes."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.Yellow +
			"Shutting down..."
		handler.CurrentStatus.ShowConnection = false
	case stateOff:
		handler.OnConnect = onConnectIdle
		handler.CurrentStatus.Message = prefixText + chat.Gold +
			"Powered off. Connect to start."
		handler.CurrentStatus.ShowConnection = true
	case stateStarting:
		connectMessage = "Sorry, the server is still starting up.\n\n" +
			"Try connecting again in a few minutes."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.LightGreen +
			"Starting up..."
		handler.CurrentStatus.ShowConnection = false
	case stateUnavailable:
		connectMessage = "The server is unavailable due to an error.\n\n" +
			"Contact the server owner for help."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.Red + "Unavailable"
		handler.CurrentStatus.ShowConnection = false
	case stateStarted:
		if currentState != stateStarted {
			stopBeacon()
			go startForwarder()
		}
	}

	currentState = s
}

func onConnectMessage(player *handler.Player) string {
	return headerText + connectMessage
}

func onConnectIdle(player *handler.Player) string {
	log.Println(player.Username + " has started the server.")
	go restoreServer()
	return headerText +
		"The server is now starting. Come back in about 5 minutes."
}

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
