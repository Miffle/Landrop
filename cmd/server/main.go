package main

import (
	webfiles "Landrop"
	"Landrop/internal/presence"
	"Landrop/internal/server"
	"log"
	"net/http"
)

// Version is set at build time via: -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	hub := presence.NewHub()
	go hub.Run()

	http.HandleFunc("/ws", server.HandleWS(hub))

	// Serve embedded static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webfiles.Static()))))

	// Serve embedded index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(webfiles.IndexHTML())
	})

	log.Printf("Landrop %s — listening on :6437", Version)
	log.Fatal(http.ListenAndServe(":6437", nil))
}
