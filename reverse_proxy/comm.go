package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"github.com/1lann/beacon/protocol"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"
)

var notifyStopped = false
var notifyChannel chan interface{}

func (s *Server) IsMinecraftServerRunning() bool {
	conn, err := net.Dial("tcp",
		s.IPAddress+":"+globalConfig.CommunicationsPort)
	if err != nil {
		s.Log("communications", "Failed to connect to remote:", err)
		return false
	}

	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(time.Second * 5))

	data, _ := ioutil.ReadAll(conn)
	if len(data) == 0 {
		s.Log("communications", "Failed to read response from remote.")
		return false
	}

	decrypted, err := decrypt(globalConfig.EncryptionKeyBytes, data)
	if err != nil {
		s.Log("communications", "Failed to decrypt response:", err)
	}

	if string(decrypted) == "started" {
		return true
	}

	return false
}

func (s *Server) IsMinecraftServerResponding() bool {
	conn, err := net.Dial("tcp", s.IPAddress+":25565")
	if err != nil {
		return false
	}

	defer conn.Close()

	stream := protocol.NewStream(conn)
	handshake := protocol.NewPacketWithId(0x00)
	handshake.WriteVarInt(5)
	handshake.WriteString(s.IPAddress)
	handshake.WriteUInt16(25565)
	handshake.WriteVarInt(1)
	if err := stream.WritePacket(handshake); err != nil {
		return false
	}

	conn.SetReadDeadline(time.Now().Add(time.Second * 5))
	_, length, err := stream.GetPacketStream()
	if err != nil {
		return false
	}

	if length > 5 {
		return true
	}

	return false
}

func (s *Server) StopMinecraftServer() {
	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp",
			s.IPAddress+":"+globalConfig.CommunicationsPort)
		if err != nil {
			s.Log("communications", "Failed to connect to remote:", err)
			continue
		}

		defer conn.Close()

		data, err := encrypt(globalConfig.EncryptionKeyBytes, []byte("stop"))
		if err != nil {
			s.Log("communications", "Failed to encrypt stop message:", err)
			return
		}

		_, err = conn.Write(data)
		if err != nil {
			s.Log("communications", "Failed to send stop message:", err)
			continue
		}

		break
	}

	notifyChannel = make(chan interface{})

	go func() {
		time.Sleep(time.Second * 30)

		if notifyStopped {
			notifyStopped = false
			close(notifyChannel)
		}
	}()

	notifyStopped = true
	<-notifyChannel
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

	decryptedData, err := decrypt(globalConfig.EncryptionKeyBytes, data)
	if err != nil {
		server.Log("communications", "Decryption failed:", err)
		return
	}

	request := string(decryptedData)
	server.Log("communications", "Received request:", request)
	switch request {
	case "started":
		server.SetState(stateStarted)
	case "stopped":
		if notifyStopped {
			notifyStopped = false
			close(notifyChannel)
			return
		}

		server.SetState(stateUnavailable)
	default:
		server.Log("communications", "Unknown request:", request)
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
