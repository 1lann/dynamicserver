package main

import (
	"github.com/digitalocean/godo"
	"time"
)

const (
	actionUnknown = iota
	actionSnapshot
	actionShuttingDown
	actionDestroy
	actionCreate
	actionErrored
	actionRunning
)

func startDropletMonitor() {
	for {
		delay := runDropletCheck()
		time.Sleep(delay)
	}
}

func runDropletCheck() (delay time.Duration) {
	delay = time.Second * 30

	serverPrefixes := make([]string, len(allServers))

	for i, server := range allServers {
		server.StateLock.Lock()
		defer server.StateLock.Unlock()
		serverPrefixes[i] = server.Name
	}

	droplets, err := getDropletsList(serverPrefixes)
	if err != nil {
		Log("droplet monitor", "Failed to get droplet list:", err)
		return time.Second * 10
	}

	for i, droplet := range droplets {
		server := allServers[i]
		if !droplet.exists {
			server.SetState(stateOff)
			continue
		}

		server.IPAddress = droplet.Networks.V4[0].IPAddress
		server.DropletId = droplet.ID

		if droplet.Status == "off" && server.State == stateShutdown {
			go server.Snapshot()
			continue
		}

		if droplet.Status == "active" && server.State == stateSnapshot {
			go server.Destroy()
			continue
		}

		if server.IsMinecraftServerRunning() {
			if server.IsMinecraftServerResponding() {
				server.SetState(stateStarted)
			} else if server.State == stateStarting {
				server.SetState(stateStarting)
			} else {
				server.SetState(stateUnavailable)
			}
			continue
		}

		event, err := getRunningAction(server, droplet.Status)
		if err != nil {
			server.Log("droplet monitor", "Failed to get running event:", err)
			server.SetState(stateUnavailable)
			continue
		}

		switch event {
		case actionUnknown:
			server.SetState(stateUnavailable)
		case actionSnapshot:
			server.SetState(stateSnapshot)
			delay = time.Second * 10
		case actionShuttingDown:
			server.SetState(stateShutdown)
			delay = time.Second * 10
		case actionDestroy:
			server.SetState(stateDestroy)
			delay = time.Second * 10
		case actionCreate:
			server.SetState(stateStarting)
			delay = time.Second * 10
		case actionErrored:
			server.SetState(stateUnavailable)
		case actionRunning:
			server.SetState(stateStopped)
		}
	}

	return delay
}

type dropletState struct {
	godo.Droplet
	exists bool
}

func getDropletsList(dropletPrefixes []string) ([]dropletState, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	var droplets []godo.Droplet

	droplets, _, err := doClient.Droplets.List(opt)
	if err == nil {
		return []dropletState{}, err
	}

	var automatedDroplets []dropletState

	for _, prefix := range dropletPrefixes {
		for _, droplet := range droplets {
			if droplet.Name == prefix+"-automated" {
				dropletInfo := dropletState{Droplet: droplet}
				dropletInfo.exists = true
				automatedDroplets = append(automatedDroplets, dropletInfo)
				break
			}
		}

		automatedDroplets = append(automatedDroplets,
			dropletState{exists: false})
	}

	return automatedDroplets, nil
}

func getRunningAction(server *Server, dropletStatus string) (int, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}
	actions, _, err := doClient.Droplets.Actions(server.DropletId, opt)
	if err != nil {
		return 0, err
	}

	// Issues with the most recent action takes priority over whether the
	// the droplet is active or not.
	if actions[0].Status == "errored" {
		return actionErrored, nil
	}

	if actions[0].Status == "completed" && dropletStatus == "active" {
		if server.State == stateShutdown {
			return actionShuttingDown, nil
		}

		return actionRunning, nil
	}

	switch actions[0].Type {
	case "create":
		return actionCreate, nil
	case "snapshot":
		if server.State != stateStopped {
			return actionSnapshot, nil
		} else {
			return actionRunning, nil
		}
	case "destroy":
		return actionDestroy, nil
	case "shutdown":
		return actionShuttingDown, nil
	}

	return actionUnknown, nil
}
