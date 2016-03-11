package main

import (
	"github.com/1lann/beacon/handler"
	"github.com/1lann/beacon/ping"
	"sync"
	"time"
)

type Server struct {
	ConfigServer
	IPAddress          string
	State              state
	PingStatus         ping.Status
	ConnectMessage     string
	StateLock          *sync.Mutex
	DropletId          int
	LastConnectionTime time.Time
	ShutdownDeadline   time.Time
	NumConnections     int
	notifyStopped      bool
	notifyChannel      chan interface{}
}

var globalConfig Config
var allServers []*Server

func main() {
	config := loadConfig()

	// Intialize the servers
	for _, server := range config.Servers {
		newServer := &Server{ConfigServer: server, StateLock: &sync.Mutex{}}

		newServer.PingStatus = ping.Status{
			MaxPlayers:     server.MaxPlayers,
			OnlinePlayers:  0,
			ProtocolNumber: server.ProtocolNumber,
		}

		if server.Available {
			newServer.SetState(stateInitializing)
		} else {
			newServer.Log("main", "Server set as not available.")
			newServer.SetState(stateUnavailable)
		}

		allServers = append(allServers, newServer)
	}

	globalConfig.CommunicationsPort = config.CommunicationsPort
	globalConfig.APIToken = config.APIToken

	watchConfig()
	loadDoClient()

	handler.OnForwardConnect = trackForwardConnect
	handler.OnForwardDisconnect = trackForwardDisconnect

	go startBeacon()
	go startDropletMonitor()
	go startConnectionMonitor()
	go startResponseMonitor()

	Log("main", "Initialized.")
	startComm()
}
