package main

import (
	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"log"
	"net/http"
)

type account struct {
	Username    string
	Password    []byte
	Permissions int
}

var socketServer *socketio.Server
var loginTemplate template.Template
var accounts []account

const authCookie = "yes u r auth'd ;)"

func loginHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth")

	if cookie.Value == authCookie {
		w.Header().Set("Location", "/dash")
		w.WriteHeader(http.StatusFound)
		return
	}

	err = r.ParseForm()
	if err != nil {
		log.Print("web: error parsing form:", err)
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	loginTemplate.Execute(w, struct{ LoginFailed bool }{LoginFailed: true})
}

func startWebInterface() {
	var err error
	socketServer, err = socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}

	loginTemplate = template.Must(template.New("login").
		ParseFiles("./web/login.html"))

	r := mux.NewRouter()
	r.HandleFunc("/socket.io", socketServer)
	// TODO: Ensure that paths work correctly
	r.HandleFunc("/login", loginHandler).Methods("POST")
	r.HandleFunc("/login", loginHandler)
	r.HandleFunc("/dash/", http.FileServer(http.Dir("./web")))
	http.Handle("/", r)
}
