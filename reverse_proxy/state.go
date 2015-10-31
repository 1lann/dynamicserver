package main

import (
	"github.com/1lann/beacon/chat"
	"github.com/1lann/beacon/handler"
	"time"
)

type state int

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

func (s *Server) SetState(st state) {
	if st != stateStarted && s.State == stateStarted {
		handler.Handle(s.Hostnames, s.ResponseHandler)
	}

	switch st {
	case stateInitializing:
		s.ConnectMessage = "Sorry, the server is not ready take requests " +
			"yet!\nTry connecting again in a few seconds."
		s.PingStatus.Message = s.Messages.MessagePrefix +
			chat.Yellow + "Initializing..."
		s.PingStatus.ShowConnection = false
	case stateStopped:
		s.ConnectMessage = "Sorry, the server is intentionally down.\n" +
			"Contact Chuie for more information."
		s.PingStatus.Message = s.Messages.MessagePrefix +
			chat.Red + "Intentionally down."
		s.PingStatus.ShowConnection = false
	case stateShutdown, stateSnapshot, stateDestroy, stateStartShutdown:
		s.ConnectMessage = "Sorry, the server is currently shutting down.\n" +
			"You may start it again when it is completely powered off."
		s.PingStatus.Message = s.Messages.MessagePrefix + chat.Yellow +
			"Shutting down..."
		s.PingStatus.ShowConnection = false
	case stateOff:
		handler.Handle(s.Hostnames, s.StartServerHandler)
		s.PingStatus.Message = s.Messages.MessagePrefix + chat.Gold +
			"Powered off. Connect to start."
		s.PingStatus.ShowConnection = true
	case stateStarting:
		s.ConnectMessage = "Sorry, the server is still starting up.\n" +
			"Try connecting again in a few minutes."
		s.PingStatus.Message = s.Messages.MessagePrefix + chat.LightGreen +
			"Starting up..."
		s.PingStatus.ShowConnection = false
	case stateUnavailable:
		s.ConnectMessage = "The server is unavailable due to an error.\n" +
			"Contact Chuie for help."
		s.PingStatus.Message = s.Messages.MessagePrefix + chat.Red +
			"Unavailable."
		s.PingStatus.ShowConnection = false
	case stateStarted:
		if s.State != stateStarted {
			s.LastConnectionTime = time.Now()
			handler.Forward(s.Hostnames, s.IPAddress+":25565")
		}
	}

	s.State = st
}
