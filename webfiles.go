package webfiles

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:web
var webFS embed.FS

// Static returns the sub-filesystem rooted at web/static.
func Static() fs.FS {
	sub, err := fs.Sub(webFS, "web/static")
	if err != nil {
		log.Fatal("webfiles: web/static not found in embed:", err)
	}
	return sub
}

// IndexHTML returns the bytes of web/templates/index.html.
func IndexHTML() []byte {
	data, err := webFS.ReadFile("web/templates/index.html")
	if err != nil {
		log.Fatal("webfiles: web/templates/index.html not found in embed:", err)
	}
	return data
}
