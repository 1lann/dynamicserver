package main

import (
	"github.com/1lann/beacon/chat"
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

const prefixText = chat.Aqua + "[Beacon] " + chat.White
const headerText = chat.Aqua + "-- [Beacon] --\n" + chat.White

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

	if s != stateStarted && currentState == stateStarted {
		stopForwarder()
		go startBeacon()
	}

	switch s {
	case stateInitializing:
		connectMessage = headerText +
			"Sorry, the server is not ready take requests yet! " +
			"Wait and try again later."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Yellow + "Initializing..."
		handler.CurrentStatus.ShowConnection = false
	case stateStopped:
		connectMessage = headerText +
			"Sorry, the server is intentionally down.\n" +
			"Contact the server owner for more information."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText +
			chat.Red + "Intentionally down"
		handler.CurrentStatus.ShowConnection = false
	case stateIdling:
		connectMessage = headerText +
			"Sorry, the server is currently shutting down. " +
			"You may start it again when it is completely turned off.\n" +
			"Try connecting again in a few minutes."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.Yellow +
			"Shutting down..."
		handler.CurrentStatus.ShowConnection = false
	case stateIdle:
		handler.OnConnect = onConnectIdle
		handler.CurrentStatus.Message = "Off"
		handler.CurrentStatus.ShowConnection = true
	case stateStarting:
		connectMessage = headerText +
			"Sorry, the server is still starting up. " +
			"Try again in a minute."
		handler.OnConnect = onConnectMessage
		handler.CurrentStatus.Message = prefixText + chat.LightGreen +
			"Starting up..."
		handler.CurrentStatus.ShowConnection = false
	case stateUnavailable:
		connectMessage = headerText +
			"The server is unavailable due to an error. " +
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
	return connectMessage
}

func onConnectIdle(player *handler.Player) string {
	log.Println(player.Username + " has started the server.")
	go restoreServer()
	return headerText + "The server is now starting. Come back in 2 minutes."
}
