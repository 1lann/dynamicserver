package main

import (
	"bufio"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

func (s *Server) IsMinecraftServerRunning() bool {
	conn, err := net.Dial("tcp",
		s.IPAddress+":"+globalConfig.CommunicationsPort)
	if err != nil {
		s.Log("communications", "Failed to connect to remote:", err)
		return false
	}

	defer conn.Close()
	reader := bufio.NewReader(conn)

	conn.SetReadDeadline(time.Now().Add(time.Second * 5))

	data, err := reader.ReadBytes('\n')
	if err != nil {
		s.Log("communications", "Failed to read response from remote:", err)
		return false
	}

	if string(data[:len(data)-1]) == "started" {
		return true
	}

	return false
}

func (s *Server) StopMinecraftServer() {
	s.TellRemote("stop")

	s.notifyChannel = make(chan interface{})

	go func() {
		time.Sleep(time.Second * 30)

		if s.notifyStopped {
			s.notifyStopped = false
			close(s.notifyChannel)
		}
	}()

	s.notifyStopped = true
	<-s.notifyChannel
}

func (s *Server) TellRemote(message string) {
	for i := 0; i < 3; i++ {
		conn, err := net.DialTimeout("tcp",
			s.IPAddress+":"+globalConfig.CommunicationsPort, time.Second*5)
		if err != nil {
			s.Log("communications", "Failed to connect to remote:", err)
			continue
		}

		defer conn.Close()

		_, err = conn.Write([]byte(message))
		if err != nil {
			s.Log("communications", "Failed to send stop message:", err)
			continue
		}

		break
	}
}

func startComm() {
	listener, err := net.Listen("tcp", ":"+globalConfig.CommunicationsPort)
	if err != nil {
		Fatal("communications", "Failed to listen:", err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			Log("communications", "Communications stopped due to an error:",
				err)
			return
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := strings.Split(conn.RemoteAddr().String(), ":")[0]

	// Check IP address
	var server *Server
	for _, checkServer := range allServers {
		if checkServer.IPAddress == remoteAddr {
			server = checkServer
			break
		}
	}

	if server == nil {
		Log("communications", "Attempted connection from unknown IP:",
			remoteAddr)
		return
	}

	if !server.Available {
		return
	}

	if server.State == stateDestroy || server.State == stateSnapshot {
		// Ignore request, you're about to be crushed!
		return
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * 5))

	data, err := ioutil.ReadAll(conn)
	if err != nil {
		server.Log("communications",
			"Error receiving request from remote:", err)
		return
	}

	request := string(data)
	server.Log("communications", "Received request:", request)
	switch request {
	case "started":
		if server.IsMinecraftServerResponding() {
			server.SetState(stateStarted)
		} else {
			server.SetState(stateStarting)
		}
	case "stopped":
		if server.notifyStopped {
			server.notifyStopped = false
			close(server.notifyChannel)
			return
		}

		server.SetState(stateUnavailable)
	default:
		server.Log("communications", "Unknown request:", request)
	}
}
