package server

import (
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

const indexFile = "index.html"

var noCacheFiles = map[string]bool{
	indexFile:              true,
	"sw.js":                true,
	"manifest.webmanifest": true,
}

type spa struct {
	files fs.FS
}

func NewSPA(dir string) http.Handler {
	return spa{files: os.DirFS(dir)}
}

func (s spa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

		return
	}

	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")

	if name == "" || !s.isFile(name) {
		name = indexFile
	}

	setCacheHeaders(w, name)

	http.ServeFileFS(w, r, s.files, name)
}

func (s spa) isFile(name string) bool {
	info, err := fs.Stat(s.files, name)

	return err == nil && info.Mode().IsRegular()
}

func setCacheHeaders(w http.ResponseWriter, name string) {
	switch {
	case strings.HasPrefix(name, "assets/"):
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	case noCacheFiles[name]:
		w.Header().Set("Cache-Control", "no-cache")
	}
}
