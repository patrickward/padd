package main

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"

	"github.com/patrickward/padd"
)

// handleImages creates a file server that serves images from both static defaults and user directory
func (s *Server) handleImages() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the image path (remove "/images/" prefix)
		imagePath := strings.TrimPrefix(r.URL.Path, "/images/")
		if imagePath == "" {
			http.NotFound(w, r)
			return
		}

		// First, try to serve from the user's images directory
		userImagePath := filepath.Join("images", imagePath)
		if s.rootManager.FileExists(userImagePath) {
			content, err := s.rootManager.ReadFile(userImagePath)
			if err == nil {
				// Set an appropriate content type
				if ext := filepath.Ext(imagePath); ext != "" {
					contentType := getImageContentType(ext)
					if contentType != "" {
						w.Header().Set("Content-Type", contentType)
					}
				}
				_, _ = w.Write(content)
				return
			}
		}

		// If not found in the user directory, try static embedded files
		staticPath := "static/images/" + imagePath
		if file, err := padd.StaticFS.Open(staticPath); err == nil {
			defer func(file fs.File) {
				_ = file.Close()
			}(file)

			if stat, err := file.Stat(); err == nil && !stat.IsDir() {
				// Set an appropriate content type
				if ext := filepath.Ext(imagePath); ext != "" {
					contentType := getImageContentType(ext)
					if contentType != "" {
						w.Header().Set("Content-Type", contentType)
					}
				}
				http.ServeContent(w, r, imagePath, stat.ModTime(), file.(io.ReadSeeker))
				return
			}
		}

		// Image wasn't found in either location
		http.NotFound(w, r)
	})
}

// handleIconsAPI serves a JSON list of available icon names from both static and user directories
func (s *Server) handleIconsAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	iconNames := make([]string, 0)
	seen := make(map[string]bool)

	// Check user's icons directory first
	userIconsDir := filepath.Join("images", "icons")
	_ = s.rootManager.WalkDir(userIconsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".svg") {
			iconName := strings.TrimSuffix(d.Name(), ".svg")
			if !seen[iconName] {
				iconNames = append(iconNames, iconName)
				seen[iconName] = true
			}
		}
		return nil
	})

	// Then check static embedded icons
	if entries, err := padd.StaticFS.ReadDir("static/images/icons"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".svg") {
				iconName := strings.TrimSuffix(entry.Name(), ".svg")
				if !seen[iconName] {
					iconNames = append(iconNames, iconName)
					seen[iconName] = true
				}
			}
		}
	}

	// Sort the icon names for consistent ordering
	sort.Strings(iconNames)

	if err := json.NewEncoder(w).Encode(iconNames); err != nil {
		s.showServerError(w, r, err)
	}
}

// getImageContentType returns the appropriate MIME type for image extensions
func getImageContentType(ext string) string {
	switch strings.ToLower(ext) {
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	default:
		return ""
	}
}
