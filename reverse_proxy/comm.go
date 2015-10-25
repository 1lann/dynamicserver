package main

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

const commPort = "9010"

func runCommDaemon() {
	listener, err := net.Listen("tcp", ":"+commPort)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleCommConnection(conn)
	}
}

func handleCommConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	data, err := reader.ReadBytes('\n')
	if err != nil {
		log.Println("Error while reading connection:", err)
		return
	}

	if strings.Trim(string(data), " \n") != token {
		log.Println("Invalid token:", string(data))
		return
	}

	data, err = reader.ReadBytes('\n')
	if err != nil && err != io.EOF {
		log.Println("Error while reading request:", err)
	}

	request := strings.Trim(string(data), " \n")
	switch request {
	case "started":
		forwardAddr = strings.Split(conn.RemoteAddr().String(), ":")[0]
		setState(stateStarted)
	case "stopped":
		setState(stateStopped)
	case "destroy":
		go shutdownServer()
	default:
		log.Println("Unknown request:" + request)
	}

	return
}

func isMinecraftServerRunning() bool {
	// Returns whether the minecraft server is running.
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", forwardAddr+":"+commPort)
		if err != nil {
			conn.Close()
			time.Sleep(time.Second)
			continue
		}

		_, err = conn.Write([]byte("status\n"))
		if err != nil {
			conn.Close()
			time.Sleep(time.Second)
			continue
		}

		reader := bufio.NewReader(conn)
		data, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			conn.Close()
			time.Sleep(time.Second)
			continue
		}

		response := strings.Trim(string(data), " \n")
		conn.Close()
		if response == "started" {
			return true
		} else {
			return false
		}
	}

	return false
}
