package main

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// getStaticDir returns EASYPOUR_STATIC_DIR if set and the directory exists; otherwise "".
func getStaticDir() string {
	dir := os.Getenv("EASYPOUR_STATIC_DIR")
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil || !isDir(abs) {
		return ""
	}
	return abs
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// newSPAHandler returns the frontend handler, or nil when no static directory is configured.
func newSPAHandler() http.Handler {
	staticDir := getStaticDir()
	if staticDir == "" {
		return nil
	}
	handler, err := spaFileServer(staticDir)
	if err != nil {
		logrus.Warnf("Serve frontend from %s: %v", staticDir, err)
		return nil
	}
	logrus.Infof("Serving frontend from %s", staticDir)
	return handler
}

// spaHandler serves files from root and falls back to index.html for GET/HEAD
// so SPA client-side routing works when the user navigates or refreshes.
type spaHandler struct {
	root  *os.Root
	files http.Handler
}

// spaFileServer confines all file access to root via os.Root, so paths cannot escape the static directory.
func spaFileServer(root string) (http.Handler, error) {
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	return &spaHandler{root: r, files: http.FileServerFS(r.FS())}, nil
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !isReadMethod(r.Method) {
		http.NotFound(w, r)
		return
	}
	if h.isStaticFile(r.URL.Path) {
		h.files.ServeHTTP(w, r)
		return
	}
	h.serveIndex(w, r)
}

func (h *spaHandler) isStaticFile(urlPath string) bool {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		return false
	}
	fi, err := h.root.Stat(name)
	return err == nil && fi.Mode().IsRegular()
}

// serveIndex writes index.html directly; http.FileServer would redirect /index.html to ./ (301),
// which breaks client-side routes on refresh.
func (h *spaHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	f, err := h.root.Open("index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil || fi.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeContent(w, r, "index.html", fi.ModTime(), f)
}
