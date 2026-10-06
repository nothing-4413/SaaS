// Package adminui embeds the demonstration management console in the API
// binary, so the product can be started with a single service endpoint.
package adminui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed static/*
var files embed.FS

func Handler() http.Handler {
	static, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Base(r.URL.Path)
		if r.URL.Path == "/" || r.URL.Path == "/reset-password" || name == "." || name == "/" {
			name = "index.html"
		}
		if name != "index.html" && name != "app.js" && name != "styles.css" && name != "favicon.svg" {
			http.NotFound(w, r)
			return
		}
		if name == "index.html" {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=3600")
		}
		content, err := fs.ReadFile(static, name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch name {
		case "styles.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case "app.js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case "favicon.svg":
			w.Header().Set("Content-Type", "image/svg+xml")
		default:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		_, _ = w.Write(content)
	})
}
