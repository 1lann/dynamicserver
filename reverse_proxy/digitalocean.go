package main

import (
	"encoding/hex"
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

const sshFingerprint = "2e:a9:01:eb:55:f1:21:9f:31:c4:12:5f:1c:16:d6:59"

var doClient *godo.Client
var doToken string
var doTokenBytes []byte
var stateLock *sync.Mutex = &sync.Mutex{}
var failureWait time.Duration = time.Second * 5

var dropletStateImmunity time.Time
var currentDropletState int

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

	doToken = strings.Trim(string(fileData), " \n")
	doTokenBytes, err = hex.DecodeString(doToken)
	if err != nil {
		log.Fatal("[Token] WARNING: Failed to decode token!")
	}

	tokenSource := &TokenSource{
		AccessToken: doToken,
	}

	oauthClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
	doClient = godo.NewClient(oauthClient)
}

func shutdownServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateStartShutdown)
	for i := 0; i < 5; i++ {
		droplet, err := getRunningDroplet()
		log.Println("[Shutdown] Attempting to shutdown:", droplet.name)
		if err != nil {
			log.Println("[Shutdown] Failed get droplet information:", err)
			time.Sleep(failureWait)
			continue
		}

		if droplet.currentState == dropletStateShuttingDown ||
			droplet.currentState == dropletStateOff {
			log.Println("[Shutdown] Droplet already shutting down!")
			return
		}

		_, _, err = doClient.DropletActions.Shutdown(droplet.id)
		if err != nil {
			log.Println("[Shutdown] Failed to shutdown droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		setState(stateShutdown)
		setImmuneState(dropletStateShuttingDown)
		log.Println("[Shutdown] Shutdown successful.")

		return
	}

	log.Println("[Shutdown] Giving up shutdown.")
	setState(stateUnavailable)
}

func snapshotServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	// Will be followed by a destruction
	for i := 0; i < 5; i++ {
		droplet, err := getRunningDroplet()
		if err != nil {
			log.Println("[Snapshot] Failed to get running droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		if droplet.currentState == dropletStateSnapshot {
			log.Println("[Snapshot] Snapshot already in progress!")
			return
		}

		snapshotTime := time.Now().Unix()
		_, _, err = doClient.DropletActions.Snapshot(droplet.id,
			"minecraft-"+strconv.FormatInt(snapshotTime, 10))
		if err != nil {
			log.Println("[Snapshot] Failed to snapshot droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		setState(stateSnapshot)
		setImmuneState(dropletStateSnapshot)
		log.Println("[Snapshot] Created snapshot: minecraft-", snapshotTime)
		break
	}

	log.Println("[Snapshot] Now deleting old snapshots.")

	for i := 0; i < 5; i++ {
		opt := &godo.ListOptions{
			Page:    1,
			PerPage: 100,
		}

		mcSnapshots := []snapshotInfo{}

		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			log.Println("[Snapshot] Failed to list snapshots:", err)
			time.Sleep(failureWait)
			continue
		}
		for _, snapshot := range snapshots {
			if len(snapshot.Name) > 10 && snapshot.Name[:10] == "minecraft-" {
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

		deletionAttempts := 0
		for len(mcSnapshots) > 3 && deletionAttempts < 5 {
			// Destroy until only 2 snapshots remain.
			deletionAttempts++

			earliestIndex := -1
			earliestSnapshot := snapshotInfo{time: time.Now().Unix(), id: 0}
			for k, snapshot := range mcSnapshots {
				if snapshot.time < earliestSnapshot.time {
					earliestIndex = k
					earliestSnapshot = snapshot
				}
			}

			if earliestIndex >= 0 {
				log.Println("[Snapshot] Removing snapshot: minecraft-",
					earliestSnapshot.time)
				_, err := doClient.Images.Delete(earliestSnapshot.id)
				if err != nil {
					log.Println("[Snapshot] Failed to remove snapshot:", err)
					time.Sleep(failureWait)
					continue
				}

				mcSnapshots[earliestIndex] = mcSnapshots[len(mcSnapshots)-1]
				mcSnapshots = mcSnapshots[:len(mcSnapshots)-1]
			}
		}

		if deletionAttempts >= 5 {
			log.Println("[Snapshot] Giving up deleting old snapshots.")
		}

		return
	}

	log.Println("[Snapshot] Giving up finding old snapshots.")
}

func destroyServer(id int) {
	stateLock.Lock()
	defer stateLock.Unlock()

	log.Println("[Destroy] Destroying droplet:", id)

	if id == 3608740 {
		log.Println("[DANGER] SAFETY CHECK FAIL: ATTEMPT TO DESTROY MAIN DROPLET.")
		return
	}

	for i := 0; i < 5; i++ {
		_, err := doClient.Droplets.Delete(id)
		if err != nil {
			log.Println("[Destroy] Error while destroying droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		setState(stateDestroy)
		setImmuneState(dropletStateDestroy)
		log.Println("[Destroy] Destroy successful.")

		return
	}

	log.Println("[Destroy] Giving up destroying.")
	setState(stateUnavailable)
}

func restoreServer() {
	stateLock.Lock()
	defer stateLock.Unlock()

	setState(stateStarting)

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	var latestSnapshot snapshotInfo

	for i := 0; i < 5; i++ {
		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			log.Println("[Restore] Failed to list snapshots:", err)
			time.Sleep(failureWait)
			continue
		}
		for _, snapshot := range snapshots {
			if len(snapshot.Name) > 10 && snapshot.Name[:10] == "minecraft-" {
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

		break
	}

	if latestSnapshot.id == 0 {
		log.Println("[Restore] No valid snapshots found!")
		return
	}

	createRequest := &godo.DropletCreateRequest{
		Name:   "minecraft-automated",
		Region: "sgp1",
		Size:   "1gb",
		Image: godo.DropletCreateImage{
			ID: latestSnapshot.id,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: sshFingerprint,
			},
		},
	}

	log.Println("[Restore] Attempting to restore snapshot minecraft-",
		latestSnapshot.time)

	for i := 0; i < 5; i++ {
		// Safety check: Make sure there aren't more than 4 droplets running.
		droplets, _, err := doClient.Droplets.List(opt)
		if err != nil {
			log.Println("[Restore] Failed to get droplet list:", err)
			time.Sleep(failureWait)
			continue
		}

		if len(droplets) > 4 {
			// Refuse to create droplet
			log.Println("[Restore] Too many existing droplets, droplet restoration cancelled.")
			return
		}

		for _, droplet := range droplets {
			if droplet.Name == "minecraft-automated" {
				log.Println("[Restore] There is an already existing " +
					"minecraft droplet. Waiting and retrying.")
				time.Sleep(time.Second * 10)
				continue
			}
		}

		_, _, err = doClient.Droplets.Create(createRequest)
		if err != nil {
			log.Println("[Restore] Failed to create droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		setImmuneState(dropletStateSnapshot)
		log.Println("[Restore] Restore successful.")

		return
	}

	log.Println("[Restore] Giving up restoring.")
}

func setImmuneState(dropletState int) {
	dropletStateImmunity = time.Now().Add(time.Second * 10)
	currentDropletState = dropletState
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
		if droplet.Name == "minecraft-automated" {
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

	forwardAddr = runningDropletInfo.ipAddress

	if time.Now().Before(dropletStateImmunity) {
		runningDropletInfo.currentState = currentDropletState
		return runningDropletInfo, nil
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

	if actions[0].Status == "completed" &&
		runningDroplet.Status == "active" {
		if currentState == stateShutdown {
			runningDropletInfo.currentState = dropletStateShuttingDown
		} else {
			runningDropletInfo.currentState = dropletStateActive
		}

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
