package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed all:build
var buildFS embed.FS

func Handler() http.Handler {
	subFS, err := fs.Sub(
		buildFS,
		"build",
	)

	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(
		http.FS(subFS),
	)

	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		requestPath := strings.TrimPrefix(
			path.Clean(r.URL.Path),
			"/",
		)

		if requestPath == "." || requestPath == "" {
			serveIndex(
				w,
				r,
				subFS,
			)
			return
		}

		file, err := subFS.Open(requestPath)

		if err == nil {
			defer file.Close()

			if stat, statErr := file.Stat(); statErr == nil && !stat.IsDir() {
				fileServer.ServeHTTP(
					w,
					r,
				)
				return
			}
		}

		serveIndex(
			w,
			r,
			subFS,
		)
	})
}

func serveIndex(
	w http.ResponseWriter,
	r *http.Request,
	files fs.FS,
) {
	index, err := fs.ReadFile(
		files,
		"index.html",
	)

	if err != nil {
		http.Error(
			w,
			"frontend index not found",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)
	w.WriteHeader(http.StatusOK)

	if r.Method == http.MethodHead {
		return
	}

	w.Write(index)
}
