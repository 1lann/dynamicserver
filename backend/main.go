package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
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

var stateStarted []byte
var stateDestory []byte
var stateStopped []byte

var lastState []byte

func main() {
	_ = os.Remove(flagFile)

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fileData, err := ioutil.ReadFile(dir + "/token.txt")
	if err != nil {
		log.Fatal("token.txt read error:", err)
	}

	token := strings.Trim(string(fileData), " \n")
	byteToken, err := hex.DecodeString(token)
	if err != nil {
		log.Fatal(err)
	}

	if stateStarted, err = encrypt(byteToken, []byte("started")); err != nil {
		log.Fatal(err)
	}
	if stateDestory, err = encrypt(byteToken, []byte("destroy")); err != nil {
		log.Fatal(err)
	}
	if stateStopped, err = encrypt(byteToken, []byte("stopped")); err != nil {
		log.Fatal(err)
	}

	if bytes.Equal(checkState(), stateStopped) {
		parsedRunCommand := strings.Fields(runCommand)

		cmd := exec.Command(parsedRunCommand[0], parsedRunCommand[1:]...)
		cmd.Dir = workingDirectory
		_ = cmd.Start()
	}

	go respondState()

	for {
		newState := checkState()
		if !bytes.Equal(newState, lastState) {
			sendState(newState, token)
			lastState = newState
		}

		time.Sleep(time.Second * 2)
	}
}

func checkState() []byte {
	parsedCheckCommand := strings.Fields(checkCommand)

	cmd := exec.Command(parsedCheckCommand[0], parsedCheckCommand[1:]...)
	response, _ := cmd.Output()
	if strings.Contains(string(response), "minecraft") {
		return stateStarted
	}

	if _, err := os.Stat(flagFile); !os.IsNotExist(err) {
		return stateDestory
	}

	return stateStopped
}

func sendState(state []byte, token string) {
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", masterAddress+":"+commPort)
		if err != nil {
			log.Println("Could not connect to master:", err)
			time.Sleep(time.Second)
			continue
		}

		_, err = conn.Write(state)
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

			_, err = conn.Write(lastState)
			if err != nil {
				log.Println("Failed to write response:", err)
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
