package main

import (
	webfiles "Landrop"
	"Landrop/internal/presence"
	"Landrop/internal/server"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Version is set at build time via: -ldflags "-X main.Version=v1.2.3"
var Version = "dev"

func main() {
	port := flag.Int("port", 6437, "Port to listen on")
	basePath := flag.String("base-path", "", "URL prefix this app is mounted at (e.g. /m/landrop)")
	flag.Parse()

	addr := fmt.Sprintf(":%d", *port)

	base := strings.TrimRight(*basePath, "/")
	if base != "" {
		base += "/"
	} else {
		base = "/"
	}

	hub := presence.NewHub()
	go hub.Run()

	mux := http.NewServeMux()

	mux.HandleFunc("/ws", server.HandleWS(hub))

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webfiles.Static()))))

	indexHTML := injectBase(webfiles.IndexHTML(), base)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	log.Printf("Landrop %s — listening on %s (base: %q)", Version, addr, base)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func injectBase(html []byte, basePath string) []byte {
	tag := []byte(fmt.Sprintf(`<base href="%s">`, basePath))
	// Try to insert after <head> (case-insensitive).
	for _, marker := range [][]byte{[]byte("<head>"), []byte("<HEAD>")} {
		if idx := bytes.Index(html, marker); idx != -1 {
			pos := idx + len(marker)
			result := make([]byte, 0, len(html)+len(tag)+1)
			result = append(result, html[:pos]...)
			result = append(result, '\n')
			result = append(result, tag...)
			result = append(result, html[pos:]...)
			return result
		}
	}
	// Fallback: prepend tag if <head> not found
	return append(tag, html...)
}
