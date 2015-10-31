package main

import (
	"time"
)

func trackForwardConnect(ipAddress string) {
	for _, server := range allServers {
		if server.IPAddress == ipAddress {
			server.NumConnections++
			break
		}
	}
}

func trackForwardDisconnect(ipAddress string) {
	for _, server := range allServers {
		if server.IPAddress == ipAddress {
			server.NumConnections--
			if server.NumConnections == 0 {
				server.Log("connection", "All players have left.")
				server.LastConnectionTime = time.Now()
			}
			break
		}
	}
}

func startConnectionMonitor() {
	for {
		for _, server := range allServers {
			checkServerConnections(server)
		}

		time.Sleep(time.Minute)
	}
}

func checkServerConnections(server *Server) {
	server.StateLock.Lock()
	defer server.StateLock.Unlock()

	if server.State == stateStarted && server.NumConnections == 0 &&
		time.Now().Sub(server.LastConnectionTime) >=
			time.Duration(server.AutoShutdownMinutes)*time.Minute {
		server.Log("connection tracker", "Auto shutdown initiated.")
		go server.Shutdown()
	}
}
