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
		dropletRunning := false
		delay := 30
		droplet, err := getRunningDroplet()

		if err == ErrNotRunning {
			setState(stateIdle)
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

		if droplet.currentState == dropletStateCreate {
			setState(stateStarting)
			delay = 10
		} else if droplet.currentState == dropletStateDestroy ||
			droplet.currentState == dropletStateSnapshot ||
			droplet.currentState == dropletStateShuttingDown {
			setState(stateIdling)
			delay = 10
		} else if droplet.currentState == dropletStateOff {
			if currentState == stateIdling {
				go snapshotServer()
			} else {
				setState(stateStopped)
			}
		} else if droplet.currentState == dropletStateUnknown {
			setState(stateUnavailable)
		} else if droplet.currentState == dropletStateActive {
			dropletRunning = true
		} else {
			log.Println("Unhandled droplet state:", droplet.currentState)
			setState(stateUnavailable)
		}

		if dropletRunning {
			if currentState == stateIdling {
				go destroyServer(droplet.id)
				stateLock.Unlock()
				time.Sleep(10 * time.Second)
				continue
			}

			forwardAddr = droplet.ipAddress

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
