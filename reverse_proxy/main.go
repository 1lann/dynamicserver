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
	NumConnections     int
}

var globalConfig Config
var allServers []*Server

func main() {
	config := loadConfig()

	// Intialize the servers
	for _, server := range config.Servers {
		newServer := &Server{ConfigServer: server, StateLock: &sync.Mutex{}}

		newServer.PingStatus = ping.Status{
			MaxPlayers:    server.MaxPlayers,
			OnlinePlayers: 0,
		}
		newServer.SetState(stateInitializing)

		allServers = append(allServers, newServer)
	}

	globalConfig.CommunicationsPort = config.CommunicationsPort
	globalConfig.EncryptionKey = config.EncryptionKey
	globalConfig.APIToken = config.APIToken
	globalConfig.EncryptionKeyBytes = config.EncryptionKeyBytes

	loadDoClient()

	handler.OnForwardConnect = trackForwardConnect
	handler.OnForwardDisconnect = trackForwardDisconnect

	go startBeacon()
	go startDropletMonitor()
	go startConnectionMonitor()
	go startResponseMonitor()
	startComm()
}
