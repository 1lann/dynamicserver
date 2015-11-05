package main

import (
	"encoding/hex"
	"encoding/json"
	"github.com/1lann/beacon/handler"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ConfigServer struct {
	Name                string   `json:"name"`
	Available           bool     `json:"available"`
	Hostnames           []string `json:"hostnames"`
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
	Whitelist []string `json:"start_whitelist"`
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
		hex.DecodeString(newConfig.EncryptionKey)
	if err != nil {
		Fatal("config", "Invalid 256-bit base16 encoded encryption key!")
	}

	return newConfig
}

func liveLoadConfig() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Log("lconfig", "Could not resolve filepath:", err)
		return
	}

	data, err := ioutil.ReadFile(dir + "/config.json")
	if err != nil {
		Log("config", "Failed to read configuration:", err)
		return
	}

	var newConfig Config
	err = json.Unmarshal(data, &newConfig)
	if err != nil {
		Log("config", "Failed to decode configuration:", err)
		return
	}

	if len(newConfig.Servers) != len(allServers) {
		Log("config", "Number of servers have changed. "+
			"You must restart the reverse proxy for changes to take place.")
		return
	}

	if newConfig.EncryptionKey != globalConfig.EncryptionKey {
		Log("config", "The encryption key has changed. You must restart "+
			"the server to use the new encryption key.")
	}

	if newConfig.APIToken != globalConfig.APIToken {
		Log("config", "The API key has changed. You must restart "+
			"the server to use the new API key.")
	}

	if newConfig.CommunicationsPort != globalConfig.CommunicationsPort {
		Log("config", "The communications port has changed. You must restart "+
			"the server to use the new communications port.")
	}

	for i, newServer := range newConfig.Servers {
		currentServer := allServers[i]

		if currentServer.Name != newServer.Name {
			Log("config", "The servers have been reordered or renamed."+
				"You must restart the reverse proxy for changes to take place.")
			continue
		}

		currentServer.Messages = newServer.Messages
		currentServer.Whitelist = newServer.Whitelist
		currentServer.Hostnames = newServer.Hostnames
		currentServer.MaxPlayers = newServer.MaxPlayers
		currentServer.Droplet = newServer.Droplet
		currentServer.AutoShutdownMinutes = newServer.AutoShutdownMinutes

		if newServer.Available {
			currentServer.Available = true
			currentServer.setStateRaw(currentServer.State)

			if currentServer.State != stateStarted &&
				currentServer.State != stateOff {
				handler.Handle(currentServer.Hostnames,
					currentServer.ResponseHandler)
			}
		} else {
			currentServer.Available = false
			currentServer.SetState(stateUnavailable)
		}
	}

	Log("config", "Reloaded configuration.")
}

func watchConfig() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		Fatal("config watcher", err)
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create {
					liveLoadConfig()
				}
			case err := <-watcher.Errors:
				Log("config watcher", err)
			}
		}
	}()

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Log("config watcher", "Could not resolve filepath:", err)
		return
	}

	err = watcher.Add(dir + "/config.json")
	if err != nil {
		Fatal("config watcher", err)
	}
}
