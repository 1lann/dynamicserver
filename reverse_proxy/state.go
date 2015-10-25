package main

import (
	"github.com/1lann/beacon/handler"
	"log"
)

type state int

var currentState state

// var lastSet time.Time

const (
	stateInitializing = iota
	stateStopped
	stateIdle
	stateIdling
	stateStarted
	stateStarting
	stateUnavailable
)

func (s state) String() string {
	switch s {
	case stateInitializing:
		return "Intializing"
	case stateStopped:
		return "Stopped"
	case stateIdle:
		return "Idle"
	case stateIdling:
		return "Idling"
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

	log.Println("[State] State is:", s)

	if s != stateStarted && currentState == stateStarted {
		stopForwarder()
		go startBeacon()
	}

	switch s {
	case stateInitializing:
		connectMessage = "Sorry, the server is not ready take requests yet! " +
			"Wait and try again later."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = "Status: Initializing..."
		handler.CurrentStatus.ShowConnection = false
	case stateStopped:
		connectMessage = "Sorry, the server is down. Try again later."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = "Status: Down"
		handler.CurrentStatus.ShowConnection = false
	case stateIdling:
		connectMessage = "The server is currently idling. " +
			"You may restart it when it is idle.\n" +
			"Try again in a minute."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = "Status: Idling..."
		handler.CurrentStatus.ShowConnection = false
	case stateIdle:
		handler.OnConnect = onConnectIdle
		handler.CurrentStatus.Message = "Status: Idle"
		handler.CurrentStatus.ShowConnection = true
	case stateStarting:
		connectMessage = "The server is still starting. Try again in a minute."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = "Status: Starting..."
		handler.CurrentStatus.ShowConnection = false
	case stateUnavailable:
		connectMessage = "The server is unavailable due to an error. " +
			"Contact the server owner for help."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = "Status: Unavailable"
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
	return connectMessage
}

func onConnectIdle(player *handler.Player) string {
	log.Println(player.Username + " has started the server.")
	go restoreServer()
	return "The server is now starting. Come back in 2 minutes."
}
