package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
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

	remoteAddr := strings.Split(conn.RemoteAddr().String(), ":")[0]
	if remoteAddr != forwardAddr {
		log.Println("[Comm] Attempted connection from unknown host:",
			remoteAddr)
		return
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * 5))

	data, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Println("[Comm] Error receiving request from remote:", err)
		return
	}

	decryptedData, err := decrypt(doTokenBytes, data)
	if err != nil {
		log.Println("[Comm] Decryption failed.")
		return
	}

	request := string(decryptedData)
	log.Println("[Comm] Receive request:", request)
	switch request {
	case "started":
		forwardAddr = strings.Split(conn.RemoteAddr().String(), ":")[0]
		setState(stateStarted)
	case "stopped":
		setState(stateStopped)
	case "destroy":
		go shutdownServer()
	default:
		log.Println("[Comm] Unknown request:" + request)
	}

	return
}

func isMinecraftServerRunning() bool {
	// Returns whether the minecraft server is running.
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", forwardAddr+":"+commPort)
		if err != nil {
			log.Println("[Comm] Error connecting to remote:", err)
			time.Sleep(time.Second)
			continue
		}

		conn.SetReadDeadline(time.Now().Add(time.Second * 5))

		data, err := ioutil.ReadAll(conn)
		if err != nil {
			log.Println("[Comm] Error pinging remote:", err)
			return false
		}

		decryptedData, err := decrypt(doTokenBytes, data)
		if err != nil {
			log.Println("[Comm] Decryption failed.")
			return false
		}

		conn.Close()

		response := string(decryptedData)
		if response == "started" {
			return true
		} else {
			return false
		}
	}

	return false
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
