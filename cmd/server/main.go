package main

import (
	"Test/internal/presence"
	"Test/internal/server"
	"log"
	"net/http"
)

// Version is set at build time via: -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	hub := presence.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", server.HandleWS(hub))

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
	})

	log.Printf("Landrop %s — listening on :6437", Version)
	log.Fatal(http.ListenAndServe(":6437", nil))
}
