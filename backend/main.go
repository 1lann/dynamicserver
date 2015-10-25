package main

import (
	"bufio"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commPort = "9010"
const masterAddress = "128.199.199.156"
const runCommand = "screen -dmS minecraft java -Xmx850M -jar /root/spigot/spigot.jar"
const keyname = "minecraft"
const workingDirectory = "/root/spigot"
const flagFile = "/root/spigot/destroy.txt"
const checkCommand = "screen -list"

var lastState = ""

func main() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fileData, err := ioutil.ReadFile(dir + "/token.txt")
	if err != nil {
		log.Fatal(err)
	}

	token := strings.Trim(string(fileData), " \n")

	parsedRunCommand := strings.Fields(runCommand)

	cmd := exec.Command(parsedRunCommand[0], parsedRunCommand[1:]...)
	cmd.Dir = workingDirectory
	_ = cmd.Start()

	go respondState()

	for {
		newState := checkState()
		if newState != lastState {
			sendState(newState, token)
			lastState = newState
		}

		time.Sleep(time.Second * 2)
	}
}

func checkState() string {
	parsedCheckCommand := strings.Fields(checkCommand)

	cmd := exec.Command(parsedCheckCommand[0], parsedCheckCommand[1:]...)
	response, _ := cmd.Output()
	if strings.Contains(string(response), "minecraft") {
		return "started"
	}

	if _, err := os.Stat(flagFile); !os.IsNotExist(err) {
		return "destroy"
	}

	return "stopped"
}

func sendState(state string, token string) {
	log.Println("Sending state:", state)
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", masterAddress+":"+commPort)
		if err != nil {
			log.Println("Could not connect to master:", err)
			time.Sleep(time.Second)
			continue
		}

		_, err = conn.Write([]byte(token + "\n" + state + "\n"))
		if err != nil {
			log.Println("Could not send message to master:", err)
			conn.Close()
			time.Sleep(time.Second)
			continue
		}

		conn.Close()
	}
}

func respondState() {
	listener, err := net.Listen("tcp", ":"+commPort)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			reader := bufio.NewReader(conn)
			data, err := reader.ReadBytes('\n')
			if err != nil && err != io.EOF {
				log.Println("Failed to read request:", err)
			}

			request := strings.Trim(string(data), " \n")
			if request != "status" {
				log.Println("Received unknown request:", request)
				return
			}

			_, err = conn.Write([]byte(lastState + "\n"))
			if err != nil {
				log.Println("Failed to write response:", err)
			}
		}(conn)
	}
}
