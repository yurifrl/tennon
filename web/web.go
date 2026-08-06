// Package web serves Tennon's embedded web UI. The Vite/React app built under
// ui/dist is embedded via go:embed and served at / with client-side routing
// fallback to index.html. API routes (/api/*, /healthz) are handled by the
// parent mux.
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/dist
var distFS embed.FS

// Handler returns the embedded web UI handler. It serves the built SPA from
// ui/dist/ with fallback to index.html for client-side routing.
func Handler() http.Handler {
	// Strip the "ui/dist" prefix to serve from the root
	stripped, err := fs.Sub(distFS, "ui/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(stripped))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// For any non-root path that doesn't exist as a file, serve index.html
		// (client-side routing)
		if r.URL.Path != "/" {
			// Try to open the file
			if _, err := stripped.Open(r.URL.Path[1:]); err != nil {
				// File doesn't exist, serve index.html for SPA routing
				r.URL.Path = "/"
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}
