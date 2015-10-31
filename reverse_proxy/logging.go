package main

import (
	"log"
)

func (s *Server) Log(module string, message ...interface{}) {
	log.Println(append([]interface{}{s.Name + " | " + module + " |"},
		message...)...)
}

func Log(module string, message ...interface{}) {
	log.Println(append([]interface{}{"general | " + module + " |"},
		message...)...)
}

func Fatal(module string, message ...interface{}) {
	log.Fatal(append([]interface{}{"general | " + module + " |"},
		message...)...)
}
