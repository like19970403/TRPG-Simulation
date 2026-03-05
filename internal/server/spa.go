package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// spaHandler serves static files from dir, falling back to index.html
// for any path that doesn't match a real file (SPA client-side routing).
type spaHandler struct {
	dir string
	fs  http.Handler
}

func newSPAHandler(dir string) *spaHandler {
	return &spaHandler{
		dir: dir,
		fs:  http.FileServer(http.Dir(dir)),
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean the path to prevent directory traversal.
	p := filepath.Clean(r.URL.Path)
	if p == "." {
		p = "/"
	}

	// Check if the file exists on disk.
	fullPath := filepath.Join(h.dir, p)
	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		// Real file exists — serve it directly with appropriate cache headers.
		switch {
		case strings.HasPrefix(p, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case p == "/sw.js" || strings.HasPrefix(p, "/workbox-"):
			w.Header().Set("Cache-Control", "no-cache")
		}
		h.fs.ServeHTTP(w, r)
		return
	}

	// File doesn't exist — serve index.html for SPA routing.
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filepath.Join(h.dir, "index.html"))
}
