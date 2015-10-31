package main

import (
	"log"
)

func (s *Server) Log(module string, message ...interface{}) {
	log.Println(s.Name+" | "+module+" |", message)
}

func Log(module string, message ...interface{}) {
	log.Println("ALL SERVERS | "+module+" |", message)
}

func Fatal(module string, message ...interface{}) {
	log.Fatal("ALL SERVERS | "+module+" |", message)
}
