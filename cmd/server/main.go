package main

import (
	webfiles "Landrop"
	"Landrop/internal/presence"
	"Landrop/internal/server"
	"flag"
	"fmt"
	"log"
	"net/http"
)

// Version is set at build time via: -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	port := flag.Int("port", 6437, "Port to listen on")
	basePath := flag.String("base-path", "", "URL prefix this app is mounted at (e.g. /m/landrop)")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)

	hub := presence.NewHub()
	go hub.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", server.HandleWS(hub))

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webfiles.Static()))))

	// Index page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(webfiles.IndexHTML())
	})

	log.Printf("Landrop %s — listening on %s (base-path: %q)", Version, addr, *basePath)
	log.Fatal(http.ListenAndServe(addr, mux))
}
