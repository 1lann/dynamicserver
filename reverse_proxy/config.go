package main

import (
	"encoding/json"
	"github.com/1lann/beacon/handler"
	"gopkg.in/fsnotify.v1"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type ConfigServer struct {
	Name                string   `json:"name"`
	Available           bool     `json:"available"`
	Hostnames           []string `json:"hostnames"`
	MaxPlayers          int      `json:"max_players"`
	ProtocolNumber      int      `json:"protocol_number"`
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
	Servers            []ConfigServer `json:"servers"`
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

	return newConfig
}

func liveLoadConfig() {
	time.Sleep(time.Second * 3)
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		Log("config", "Could not resolve filepath:", err)
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
		currentServer.ProtocolNumber = newServer.ProtocolNumber
		currentServer.Droplet = newServer.Droplet
		currentServer.AutoShutdownMinutes = newServer.AutoShutdownMinutes

		currentServer.PingStatus.MaxPlayers = currentServer.MaxPlayers
		currentServer.PingStatus.ProtocolNumber = currentServer.ProtocolNumber

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
