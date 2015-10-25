package main

import (
	"github.com/1lann/beacon/chat"
	"github.com/1lann/beacon/handler"
	"github.com/1lann/beacon/ping"
	"log"
)

func main() {
	handler.OnConnect = onConnect
	handler.CurrentStatus = ping.Status{
		OnlinePlayers: 0,
		MaxPlayers:    30,
		Message:       chat.Aqua + "Chan, you're shit!",
	}
	log.Println("Listening...")
	handler.Listen()
}

func onConnect(player *handler.Player) string {
	log.Println(player.Username + " attempted to connect!")
	return chat.Green + "Hi, " + player.Username + ". Your IP address is " + player.IP + "!"
}
