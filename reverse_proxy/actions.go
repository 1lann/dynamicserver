package main

import (
	"github.com/digitalocean/godo"
	"strconv"
	"time"
)

var failureWait = time.Second * 5

func (s *Server) Shutdown() {
	s.Log("shutdown", "Shutting down server...")
	s.SetState(stateShutdown)
	s.StopMinecraftServer()
	s.SetState(stateShutdown)
	s.TellRemote("shutdown")
	s.Log("shutdown", "Shutdown complete.")
}

func (s *Server) Destroy() {
	s.StateLock.Lock()
	defer func() {
		time.Sleep(time.Second * 10)
		s.StateLock.Unlock()
	}()

	s.Log("destroy", "Destroying droplet:", s.DropletId)
	s.SetState(stateDestroy)

	if s.DropletId == 3608740 {
		s.Log("destroy", "SAFETY CHECK FAIL: ATTEMPT TO DESTROY MAIN DROPLET.")
		return
	}

	for i := 0; i < 3; i++ {
		_, err := doClient.Droplets.Delete(s.DropletId)
		if err != nil {
			s.Log("destroy", "Error while destroying droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		s.Log("destroy", "Destroy successful.")
		return
	}

	s.Log("destroy", "Giving up destroying.")
	s.SetState(stateUnavailable)
}

type snapshotInfo struct {
	time int64
	id   int
}

func (s *Server) Snapshot() {
	s.StateLock.Lock()
	defer func() {
		time.Sleep(time.Second * 10)
		s.StateLock.Unlock()
	}()

	s.SetState(stateSnapshot)

	// Will be followed by a destruction
	for i := 0; i < 3; i++ {
		snapshotTime := time.Now().Unix()
		_, _, err := doClient.DropletActions.Snapshot(s.DropletId,
			s.Name+"-"+strconv.FormatInt(snapshotTime, 10))
		if err != nil {
			s.Log("snapshot", "Failed to snapshot droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		s.Log("snapshot", "Created snapshot with time:", snapshotTime)
		break
	}

	s.Log("snapshot", "Now deleting old snapshots.")

	for i := 0; i < 3; i++ {
		opt := &godo.ListOptions{
			Page:    1,
			PerPage: 100,
		}

		mcSnapshots := []snapshotInfo{}

		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			s.Log("snapshot", "Failed to list snapshots:", err)
			time.Sleep(failureWait)
			continue
		}

		for _, snapshot := range snapshots {
			prefixLength := len(s.Name) + 1
			if len(snapshot.Name) > prefixLength &&
				snapshot.Name[:prefixLength] == s.Name+"-" {
				value, err := strconv.ParseInt(snapshot.Name[prefixLength:],
					10, 64)
				if err != nil {
					s.Log("snapshot", "Failed to parse snapshot name: "+
						snapshot.Name)
					continue
				}

				mcSnapshots = append(mcSnapshots,
					snapshotInfo{id: snapshot.ID, time: value})
			}
		}

		deletionAttempts := 0
		for len(mcSnapshots) > 2 && deletionAttempts < 5 {
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
				s.Log("snapshot", "Removing snapshot with time:",
					earliestSnapshot.time)
				_, err := doClient.Images.Delete(earliestSnapshot.id)
				if err != nil {
					s.Log("snapshot", "Failed to remove snapshot:", err)
					time.Sleep(failureWait)
					continue
				}

				mcSnapshots[earliestIndex] = mcSnapshots[len(mcSnapshots)-1]
				mcSnapshots = mcSnapshots[:len(mcSnapshots)-1]
			}
		}

		if deletionAttempts >= 5 {
			s.Log("snapshot", "Giving up deleting old snapshots.")
		}

		return
	}

	s.Log("snapshot", "Giving up finding old snapshots.")
}

func (s *Server) Restore() {
	s.StateLock.Lock()
	defer func() {
		time.Sleep(time.Second * 10)
		s.StateLock.Unlock()
	}()

	s.SetState(stateStarting)

	opt := &godo.ListOptions{
		Page:    1,
		PerPage: 100,
	}

	var latestSnapshot snapshotInfo

	for i := 0; i < 5; i++ {
		snapshots, _, err := doClient.Images.ListUser(opt)
		if err != nil {
			s.Log("restore", "Failed to list snapshots:", err)
			time.Sleep(failureWait)
			continue
		}
		for _, snapshot := range snapshots {
			prefixLength := len(s.Name) + 1
			if len(snapshot.Name) > prefixLength &&
				snapshot.Name[:prefixLength] == s.Name+"-" {
				value, err := strconv.ParseInt(snapshot.Name[prefixLength:],
					10, 64)
				if err != nil {
					s.Log("restore", "Failed to parse snapshot name: "+
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
		s.Log("restore", "No valid snapshots found!")
		return
	}

	createRequest := &godo.DropletCreateRequest{
		Name:   s.Name + "-automated",
		Region: s.Droplet.Region,
		Size:   s.Droplet.Memory,
		Image: godo.DropletCreateImage{
			ID: latestSnapshot.id,
		},
		SSHKeys: []godo.DropletCreateSSHKey{
			godo.DropletCreateSSHKey{
				Fingerprint: s.Droplet.SSHFingerprint,
			},
		},
	}

	s.Log("restore", "Attempting to restore snapshot with time:",
		latestSnapshot.time)

	for i := 0; i < 3; i++ {
		droplets, _, err := doClient.Droplets.List(opt)
		if err != nil {
			s.Log("restore", "Failed to get droplet list:", err)
			time.Sleep(failureWait)
			continue
		}

		for _, droplet := range droplets {
			if droplet.Name == s.Name+"-automated" {
				s.Log("restore", "There is an already existing "+
					s.Name+" droplet. Waiting and retrying.")
				time.Sleep(time.Second * 10)
				continue
			}
		}

		_, _, err = doClient.Droplets.Create(createRequest)
		if err != nil {
			s.Log("restore", "Failed to create droplet:", err)
			time.Sleep(failureWait)
			continue
		}

		s.Log("restore", "Restore successful.")

		return
	}

	s.Log("restore", "Giving up restoring.")
}
