package main

import (
	"github.com/1lann/beacon/chat"
	"github.com/1lann/beacon/handler"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type state int

var currentState state

const (
	stateInitializing = iota
	stateStopped
	stateOff
	stateStartShutdown
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
	case stateStartShutdown:
		return "Start Shutdown"
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
		connectMessage = "Sorry, the server is not ready take requests yet!\n" +
			"Try connecting again in a few seconds."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Yellow + "Initializing..."
		handler.CurrentStatus.ShowConnection = false
	case stateStopped:
		connectMessage = "Sorry, the server is intentionally down.\n" +
			"Contact the server owner for more information."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Red + "Intentionally down"
		handler.CurrentStatus.ShowConnection = false
	case stateShutdown, stateSnapshot, stateDestroy, stateStartShutdown:
		connectMessage = "Sorry, the server is currently shutting down.\n" +
			"You may start it again when it is completely powered off."
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
		connectMessage = "Sorry, the server is still starting up.\n" +
			"Try connecting again in a few minutes."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.LightGreen +
			"Starting up..."
		handler.CurrentStatus.ShowConnection = false
	case stateUnavailable:
		connectMessage = "The server is unavailable due to an error.\n" +
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

var whitelist map[string]bool

func loadWhitelist() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fileData, err := ioutil.ReadFile(dir + "/whitelist.txt")
	if err != nil {
		log.Fatal("whitelist.txt read error:", err)
	}

	usernames := strings.Split(string(fileData), "\n")
	for _, username := range usernames {
		if username != "" {
			whitelist[normalizeName(username)] = true
		}
	}
}

func normalizeName(name string) string {
	return strings.ToLower(strings.Replace(name, " ", "", -1))
}

func onConnectMessage(player *handler.Player) string {
	return headerText + connectMessage
}

func onConnectIdle(player *handler.Player) string {
	if _, found := whitelist[normalizeName(player.Username)]; !found {
		log.Println(player.Username +
			" is not whitelisted and attempted to start the server.")
		return headerText +
			"Sorry, you are not whitelisted to start the server!"
	}

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

	err := handler.Listen("25565")
	if err != nil {
		log.Fatal(err)
	}
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

	for {
		conn, err := forwardListener.Accept()
		if err != nil {
			if strings.Contains(err.Error(),
				"use of closed network connection") || err == io.EOF {
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
