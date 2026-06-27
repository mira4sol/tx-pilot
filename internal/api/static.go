package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func registerWebUI(r chi.Router, webDir string, logger *zap.Logger) {
	if webDir == "" {
		return
	}

	indexPath := filepath.Join(webDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		if logger != nil {
			logger.Info("web dashboard not mounted", zap.String("dir", webDir), zap.Error(err))
		}
		return
	}

	if logger != nil {
		logger.Info("serving web dashboard", zap.String("dir", webDir))
	}

	r.Handle("/*", spaFileServer(webDir))
}

func spaFileServer(webDir string) http.Handler {
	fileServer := http.FileServer(http.Dir(webDir))
	indexPath := filepath.Join(webDir, "index.html")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}

		path := r.URL.Path
		if path == "/" {
			http.ServeFile(w, r, indexPath)
			return
		}

		fullPath, ok := safeWebPath(webDir, path)
		if !ok {
			http.NotFound(w, r)
			return
		}

		info, err := os.Stat(fullPath)
		if err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, indexPath)
	})
}

func safeWebPath(webDir, reqPath string) (string, bool) {
	clean := filepath.Clean(reqPath)
	if clean == "." || strings.HasPrefix(clean, "..") {
		return "", false
	}

	full := filepath.Join(webDir, clean)
	rel, err := filepath.Rel(webDir, full)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", false
	}
	return full, true
}
