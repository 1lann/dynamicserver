package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	stateStarted = "started"
	stateStopped = "stopped"
)

const version = "0.1"

var currentState string
var config Config
var isStopping bool

type Config struct {
	Check struct {
		Command  string `json:"command"`
		Contains string `json:"contains"`
	} `json:"check"`
	MasterAddress      string `json:"master_address"`
	CommunicationsPort string `json:"communications_port"`
	StartCommand       string `json:"start_command"`
	StopCommand        string `json:"stop_command"`
	ShutdownCommand    string `json:"shutdown_command"`
	WorkingDirectory   string `json:"working_directory"`
}

func main() {
	config = loadConfig()
	if checkState() == stateStopped {
		startServer()
	}

	log.Println("Initialized dynamicserver backend v" + version + ".")

	go respondState()

	for {
		newState := checkState()
		if newState != currentState {
			currentState = newState
			sendState()
		}
		time.Sleep(time.Second * 2)
	}
}

func loadConfig() Config {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadFile(dir + "/config.json")
	if err != nil {
		log.Fatal(err)
	}

	var loadedConfig Config
	err = json.Unmarshal(data, &loadedConfig)
	if err != nil {
		log.Fatal(err)
	}

	return loadedConfig
}

func checkState() string {
	checkCommand := strings.Fields(config.Check.Command)

	cmd := exec.Command(checkCommand[0], checkCommand[1:]...)
	cmd.Dir = config.WorkingDirectory
	response, _ := cmd.Output()
	if strings.Contains(string(response), config.Check.Contains) {
		return stateStarted
	}

	return stateStopped
}

func startServer() {
	log.Println("Executing command:", config.StartCommand)
	runCommand := strings.Fields(config.StartCommand)

	cmd := exec.Command(runCommand[0], runCommand[1:]...)
	cmd.Dir = config.WorkingDirectory
	_ = cmd.Start()
}

func stopServer() {
	log.Println("Executing command:", config.StopCommand)
	stopCommand := strings.Fields(config.StopCommand)

	cmd := exec.Command(stopCommand[0], stopCommand[1:]...)
	cmd.Dir = config.WorkingDirectory
	_ = cmd.Start()
}

func shutdownServer() {
	log.Println("Executing command:", config.ShutdownCommand)
	shutdownCommand := strings.Fields(config.ShutdownCommand)

	cmd := exec.Command(shutdownCommand[0], shutdownCommand[1:]...)
	_ = cmd.Start()
}

func sendState() {
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", config.MasterAddress+":"+
			config.CommunicationsPort)
		if err != nil {
			log.Println("Could not connect to master:", err)
			time.Sleep(time.Second)
			continue
		}

		_, err = conn.Write([]byte(currentState))
		if err != nil {
			log.Println("Could not send message to master:", err)
			conn.Close()
			time.Sleep(time.Second)
			continue
		}

		conn.Close()
		return
	}
}

func respondState() {
	listener, err := net.Listen("tcp", ":"+config.CommunicationsPort)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		if conn.RemoteAddr().String() != config.MasterAddress {
			log.Println("Received connection from unknown IP:",
				conn.RemoteAddr().String())
			conn.Close()
			continue
		}

		go func(conn net.Conn) {
			defer conn.Close()

			_, err = conn.Write([]byte(currentState + "\n"))
			if err != nil {
				log.Println("Failed to write response:", err)
				return
			}

			conn.SetReadDeadline(time.Now().Add(time.Second * 5))
			command, _ := ioutil.ReadAll(conn)
			if len(command) == 0 {
				return
			}

			if string(command) == "stop" {
				log.Println("Received request to stop.")
				stopServer()
			} else if string(command) == "shutdown" {
				log.Println("Received request to shutdown.")
				shutdownServer()
			} else {
				log.Println("Received unknown command:", command)
			}
		}(conn)
	}
}
