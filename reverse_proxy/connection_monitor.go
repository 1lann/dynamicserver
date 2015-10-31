package main

import (
	"strings"
	"time"
)

func trackForwardConnect(ipAddress string) {
	resolvedIP := strings.Split(ipAddress, ":")[0]
	for _, server := range allServers {
		if server.IPAddress == resolvedIP {
			server.NumConnections++
			break
		}
	}
}

func trackForwardDisconnect(ipAddress string, duration time.Duration) {
	resolvedIP := strings.Split(ipAddress, ":")[0]
	for _, server := range allServers {
		if server.IPAddress == resolvedIP {
			if duration > time.Second*20 {
				server.LastConnectionTime = time.Now()
			}
			server.NumConnections--
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
