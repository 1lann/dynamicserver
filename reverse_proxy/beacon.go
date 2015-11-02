package main

import (
	"github.com/1lann/beacon/chat"
	"github.com/1lann/beacon/handler"
	"log"
	"strings"
)

func startBeacon() {
	for _, server := range allServers {
		handler.SetStatus(server.Hostnames, &(server.PingStatus))
		handler.Handle(server.Hostnames, server.ResponseHandler)
	}

	log.Fatal(handler.Listen("25565"))
}

func (s *Server) StartServerHandler(player *handler.Player) string {
	whitelisted := false

	if len(s.Whitelist) > 0 {
		for _, whitelisedName := range s.Whitelist {
			if strings.EqualFold(whitelisedName, player.Username) {
				whitelisted = true
				break
			}
		}
	} else {
		whitelisted = true
	}

	if !whitelisted {
		s.Log("beacon", player.Username+
			" is not whitelisted and attempted to start the server.")
		return chat.Format(s.Messages.MessagePrefix) +
			"Sorry, you are not whitelisted to start the server!"
	}

	s.Log(player.Username + " started the server.")

	go s.Restore()

	return chat.Format(s.Messages.MessagePrefix) +
		"The server is now starting. Come back in about " +
		chat.Format(s.Messages.BootTime) + "."

}

func (s *Server) ResponseHandler(player *handler.Player) string {
	return chat.Format(s.Messages.MessagePrefix + s.ConnectMessage)
}
