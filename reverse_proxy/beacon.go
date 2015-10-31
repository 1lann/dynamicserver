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
	for _, whitelisedName := range s.Whitelist {
		if strings.EqualFold(whitelisedName, player.Username) {
			whitelisted = true
			break
		}
	}

	if !whitelisted {
		s.Log("beacon", player.Username+
			" is not whitelisted and attempted to start the server.")
		return chat.Format(s.Messages.MessagePrefix) +
			"Sorry, you are not whitelisted to start the server!"
	}

	s.Restore()

	return s.Messages.MessagePrefix +
		"The server is now starting. Come back in about " +
		chat.Format(s.Messages.BootTime) + "."

}

// func responseDispatcher(ipAddress string, player *handler.Player) string {
// 	for _, server := range allServers {
// 		if server.IPAddress == ipAddress {
// 			return server.ResponseHandler(player)
// 		}
// 	}

// 	return "[ Reverse Proxy Error ]\n\n" +
// 		"Sorry, the server at this hostname could not be found."
// }

func (s *Server) ResponseHandler(player *handler.Player) string {
	return chat.Format(s.Messages.MessagePrefix + s.ConnectMessage)
}
