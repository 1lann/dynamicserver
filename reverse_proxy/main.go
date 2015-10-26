package main

import (
	"github.com/1lann/beacon/handler"
	"github.com/1lann/beacon/ping"
	"log"
	"time"
)

var DestroyNext bool

func main() {
	handler.CurrentStatus = ping.Status{
		OnlinePlayers: 0,
		MaxPlayers:    30,
	}

	go startBeacon()
	setState(stateInitializing)
	loadDoClient()

	go monitorServer()
	runCommDaemon()
}

func monitorServer() {
	for {
		stateLock.Lock()
		delay := 30
		droplet, err := getRunningDroplet()

		if err == ErrNotRunning {
			setState(stateOff)
			stateLock.Unlock()
			time.Sleep(time.Second * 30)
			continue
		}

		if err != nil {
			setState(stateUnavailable)
			stateLock.Unlock()
			time.Sleep(time.Second * 30)
			continue
		}

		switch droplet.currentState {
		case dropletStateCreate:
			setState(stateStarting)
			delay = 10
		case dropletStateDestroy:
			setState(stateDestroy)
		case dropletStateSnapshot:
			setState(stateSnapshot)
			delay = 30
		case dropletStateShuttingDown:
			setState(stateShutdown)
			delay = 10
		case dropletStateOff:
			if currentState == stateShutdown {
				go snapshotServer()
			} else {
				setState(stateStopped)
			}
		case dropletStateUnknown:
			setState(stateUnavailable)
		default:
			log.Println("Unhandled droplet state:", droplet.currentState)
			setState(stateUnavailable)
		}

		if droplet.currentState == dropletStateActive {
			if currentState == stateSnapshot {
				go destroyServer(droplet.id)
				stateLock.Unlock()
				time.Sleep(10 * time.Second)
				continue
			}

			if isMinecraftServerRunning() {
				setState(stateStarted)
			} else {
				setState(stateStopped)
			}
		}

		stateLock.Unlock()
		time.Sleep(time.Second * time.Duration(delay))
	}
}
