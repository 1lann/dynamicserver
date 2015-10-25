package main

import (
	"errors"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var doClient *godo.Client
var token string
var stateLock *sync.Mutex = &sync.Mutex{}

var ErrNotRunning = errors.New("dynamicserver: server not running")
var ErrUnexpected = errors.New("dynamicserver: unexpected error")
var ErrSafetyCheckFail = errors.New("dynamicserver: failed safety check")

const (
	dropletStateUnknown = iota
	dropletStateActive
	dropletStateSnapshot
	dropletStateShuttingDown
	dropletStateDestroy
	dropletStateCreate
	dropletStateOff
)

type dropletInfo struct {
	id           int
	name         string
	ipAddress    string
	currentState int
}

type snapshotInfo struct {
	time int64
	id   int
}

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func loadDoClient() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}

	fileData, err := ioutil.ReadFile(dir + "/token.txt")
	if err != nil {
		log.Fatal(err)
	}

	token = strings.Trim(string(fileData), " \n")

	tokenSource := &TokenSource{
		AccessToken: token,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	doClient = godo.NewClient(oauthClient)
}

func shutdownServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateIdling)
	for i := 0; i < 5; i++ {
		droplet, err := getRunningDroplet()
		log.Println("[Shutdown] Attempting to shutdown:", droplet.name)
		if err != nil {
			log.Println("[Shutdown] Failed get droplet information:", err)
			time.Sleep(time.Second * 5)
			continue
		}

		_, _, err = doClient.DropletActions.Shutdown(droplet.id)
		if err != nil {
			log.Println("[Shutdown] Failed to shutdown droplet:", err)
			time.Sleep(time.Second * 5)
			continue
		}

		return
	}
}

func snapshotServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateIdling)
	// Will be followed by a destruction
	for i := 0; i < 5; i++ {
		droplet, err := getRunningDroplet()
		if err != nil {
			log.Println("[Snapshot] Failed to get running droplet:", err)
			continue
		}

		snapshotTime := time.Now().Unix()
		_, _, err = doClient.DropletActions.Snapshot(droplet.id,
			"minecraft_"+strconv.FormatInt(snapshotTime, 10))
		if err != nil {
			log.Println("[Snapshot] Failed to snapshot droplet:", err)
			continue
		}

		log.Println("[Snapshot] Created snapshot: minecraft_", snapshotTime)

		opt := &godo.ListOptions{
			Page:    1,
			PerPage: 100,
		}

		mcSnapshots := []snapshotInfo{}

		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			log.Println("[Snapshot] Failed to list snapshots:", err)
			return
		}
		for _, snapshot := range snapshots {
			if len(snapshot.Name) > 10 && snapshot.Name[:10] == "minecraft_" {
				value, err := strconv.ParseInt(snapshot.Name[10:], 10, 64)
				if err != nil {
					log.Println("[Snapshot] Failed to parse snapshot name: " +
						snapshot.Name)
					continue
				}

				mcSnapshots = append(mcSnapshots,
					snapshotInfo{id: snapshot.ID, time: value})
			}
		}

		for len(mcSnapshots) > 3 {
			// Destroy until only 2 snapshots remain.

			earliestIndex := -1
			earliestSnapshot := snapshotInfo{time: time.Now().Unix(), id: 0}
			for k, snapshot := range mcSnapshots {
				if snapshot.time < earliestSnapshot.time {
					earliestIndex = k
					earliestSnapshot = snapshot
				}
			}

			if earliestIndex >= 0 {
				log.Println("Removing snapshot: minecraft_",
					earliestSnapshot.time)
				_, err := doClient.Images.Delete(earliestSnapshot.id)
				if err != nil {
					log.Println("Failed to remove snapshot:", err)
					break
				}

				mcSnapshots[earliestIndex] = mcSnapshots[len(mcSnapshots)-1]
				mcSnapshots = mcSnapshots[:len(mcSnapshots)-1]
			}
		}

		return
	}

	log.Println("[Snapshot] Failed to snapshot server!")
}

func destroyServer(id int) {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateIdling)
	log.Println("[Destroy] Destroying droplet:", id)

	if id == 3608740 {
		log.Println("[DANGER] SAFETY CHECK FAIL: ATTEMPT TO DESTROY MAIN DROPLET.")
		return
	}

	for i := 0; i < 5; i++ {
		_, err := doClient.Droplets.Delete(id)
		if err != nil {
			log.Println("[Destroy] Error while destroying droplet:", err)
			time.Sleep(time.Second * 5)
			continue
		}

		return
	}
}

func restoreServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateStarting)

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	for i := 0; i < 5; i++ {
		// Safety check: Make sure there aren't more than 4 droplets running.
		droplets, _, err := doClient.Droplets.List(opt)
		if err != nil {
			log.Println("[Restore] Failed to get droplet list:", err)
			time.Sleep(time.Second * 5)
			continue
		}

		if len(droplets) > 4 {
			// Refuse to create droplet
			log.Println("[Restore] Too many existing droplets, droplet restoration cancelled.")
			return
		} else {
			break
		}
	}

	var latestSnapshot snapshotInfo

	for i := 0; i < 5; i++ {
		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			log.Println("[Restore] Failed to list snapshots:", err)
			time.Sleep(time.Second * 5)
			continue
		}
		for _, snapshot := range snapshots {
			if len(snapshot.Name) > 10 && snapshot.Name[:10] == "minecraft_" {
				value, err := strconv.ParseInt(snapshot.Name[10:], 10, 64)
				if err != nil {
					log.Println("[Restore] Failed to parse snapshot name: " +
						snapshot.Name)
					continue
				}

				if value > latestSnapshot.time {
					latestSnapshot = snapshotInfo{id: snapshot.ID, time: value}
				}
			}
		}
	}

	if latestSnapshot.id == 0 {
		log.Println("[Restore] No valid snapshots found!")
		return
	}

	createRequest := &godo.DropletCreateRequest{
		Name:   "minecraft_" + strconv.FormatInt(time.Now().Unix(), 10),
		Region: "sgp1",
		Size:   "1gb",
		Image: godo.DropletCreateImage{
			ID: latestSnapshot.id,
		},
	}

	for i := 0; i < 5; i++ {
		_, _, err := doClient.Droplets.Create(createRequest)
		if err != nil {
			log.Println("[Restore] Failed to create droplet:", err)
			time.Sleep(time.Second * 5)
			continue
		}
	}
}

func getRunningDroplet() (dropletInfo, error) {
	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	droplets, _, err := doClient.Droplets.List(opt)
	if err != nil {
		return dropletInfo{}, err
	}

	var runningDroplet godo.Droplet

	for _, droplet := range droplets {
		if len(droplet.Name) > 10 && droplet.Name[:10] == "minecraft_" {
			runningDroplet = droplet
			break
		}
	}

	if runningDroplet.ID == 0 {
		return dropletInfo{}, ErrNotRunning
	}

	runningDropletInfo := dropletInfo{
		id:        runningDroplet.ID,
		name:      runningDroplet.Name,
		ipAddress: runningDroplet.Networks.V4[0].IPAddress,
	}

	actions, _, err := doClient.Droplets.Actions(runningDroplet.ID, opt)
	if err != nil {
		return dropletInfo{}, err
	}

	// Issues with the most recent action takes priority over whether the
	// the droplet is active or not.
	if actions[0].Status == "errored" {
		return runningDropletInfo, ErrUnexpected
	}

	if runningDroplet.Status == "active" {
		runningDropletInfo.currentState = dropletStateActive
		return runningDropletInfo, nil
	}

	if actions[0].Status == "completed" {
		runningDropletInfo.currentState = dropletStateOff
		return runningDropletInfo, nil
	}

	switch actions[0].Type {
	case "create":
		runningDropletInfo.currentState = dropletStateCreate
	case "snapshot":
		runningDropletInfo.currentState = dropletStateSnapshot
	case "destroy":
		runningDropletInfo.currentState = dropletStateDestroy
	case "shutdown":
		runningDropletInfo.currentState = dropletStateShuttingDown
	default:
		runningDropletInfo.currentState = dropletStateUnknown
	}

	return runningDropletInfo, nil
}
