package main

import (
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ConfigServer struct {
	Name                string   `json:"name"`
	Hostnames           []string `json:"hostname"`
	MaxPlayers          int      `json:"max_players"`
	AutoShutdownMinutes int      `json:"auto_shutdown_minutes"`
	Droplet             struct {
		Memory         string `json:"memory"`
		Region         string `json:"region"`
		SSHFingerprint string `json:"ssh_fingerprint"`
	} `json:"droplet"`
	Messages struct {
		MessagePrefix    string `json:"message_prefix"`
		Owner            string `json:"owner"`
		ServerInfoPrefix string `json:"server_info_prefix"`
		BootTime         string `json:"boot_time"`
	} `json:"messages"`
	Whitelist []string
}

type Config struct {
	APIToken           string         `json:"api_token"`
	CommunicationsPort string         `json:"communications_port"`
	EncryptionKey      string         `json:"encryption_key"`
	Servers            []ConfigServer `json:"servers"`
	EncryptionKeyBytes []byte         `json:"-"`
}

func loadConfig() Config {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Fatal("config", "Could not resolve filepath:", err)
	}

	data, err := ioutil.ReadFile(dir + "/config.json")
	if err != nil {
		Fatal("config", "Failed to read configuration:", err)
	}

	var newConfig Config
	err = json.Unmarshal(data, &newConfig)
	if err != nil {
		Fatal("config", "Failed to decode configuration:", err)
	}

	newConfig.EncryptionKeyBytes, err =
		hex.DecodeString(globalConfig.EncryptionKey)
	if err != nil {
		Fatal("config", "Invalid 256-bit base16 encoded encryption key!")
	}

	return newConfig
}
