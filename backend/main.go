package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
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
	EncryptionKey      string `json:"encryption_key"`
	StartCommand       string `json:"start_command"`
	StopCommand        string `json:"stop_command"`
	ShutdownCommand    string `json:"shutdown_command"`
	WorkingDirectory   string `json:"working_directory"`
	EncryptionKeyBytes []byte `json:"-"`
}

func main() {
	config = loadConfig()
	if checkState() == stateStopped {
		startServer()
	}

	go respondState()

	go func() {
		for {
			newState := checkState()
			if newState != currentState {
				currentState = newState
				sendState()
			}
			time.Sleep(time.Second * 2)
		}
	}()

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Stopping from HTTP")
		stopServer()
	})

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Starting from HTTP")
		startServer()
	})

	http.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Shutdown from HTTP")
		shutdownServer()
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
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

	loadedConfig.EncryptionKeyBytes, err =
		hex.DecodeString(loadedConfig.EncryptionKey)
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
	runCommand := strings.Fields(config.StartCommand)

	cmd := exec.Command(runCommand[0], runCommand[1:]...)
	cmd.Dir = config.WorkingDirectory
	_ = cmd.Start()
}

func stopServer() {
	stopCommand := strings.Fields(config.StopCommand)

	cmd := exec.Command(stopCommand[0], stopCommand[1:]...)
	cmd.Dir = config.WorkingDirectory
	_ = cmd.Start()
}

func shutdownServer() {
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

		data, err := encrypt(config.EncryptionKeyBytes, []byte(currentState))
		if err != nil {
			log.Println("Could not encrypt message:", err)
			return
		}

		_, err = conn.Write(data)
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

		go func(conn net.Conn) {
			defer conn.Close()

			data, err := encrypt(config.EncryptionKeyBytes,
				[]byte(currentState))
			if err != nil {
				log.Println("Could not encrypt message:", err)
				return
			}

			_, err = conn.Write(append(data, '\n'))
			if err != nil {
				log.Println("Failed to write response:", err)
				return
			}

			conn.SetReadDeadline(time.Now().Add(time.Second * 5))
			command, err := ioutil.ReadAll(conn)
			if len(command) == 0 {
				return
			}

			decrypted, err := decrypt(config.EncryptionKeyBytes, command)
			if err != nil {
				log.Println("Failed to decrypt message from master:", err)
				return
			}

			if string(decrypted) == "stop" {
				log.Println("Received request to stop.")
				stopServer()
			} else if string(decrypted) == "shutdown" {
				log.Println("Received request to shutdown.")
				shutdownServer()
			} else {
				log.Println("Received unknown command:", decrypted)
			}
		}(conn)
	}
}

// Encrypt and decrypt functions from
// http://stackoverflow.com/questions/18817336/golang-encrypting-a-string-with-aes-and-base64
func encrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	b := base64.StdEncoding.EncodeToString(text)
	ciphertext := make([]byte, aes.BlockSize+len(b))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cfb := cipher.NewCFBEncrypter(block, iv)
	cfb.XORKeyStream(ciphertext[aes.BlockSize:], []byte(b))
	return ciphertext, nil
}

func decrypt(key, text []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(text) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := text[:aes.BlockSize]
	text = text[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(text, text)
	data, err := base64.StdEncoding.DecodeString(string(text))
	if err != nil {
		return nil, err
	}
	return data, nil
}
