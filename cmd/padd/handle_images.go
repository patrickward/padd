package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

type ImageUploadResponse struct {
	Success bool   `json:"success"`
	DataURI string `json:"dataUri,omitempty"`
	Error   string `json:"error,omitempty"`
}

// handleImageUpload handles image uploads
func (s *Server) handleImageUpload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Limit the size of uplaoded images
	const maxFileSize = 10 << 20 // 10 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxFileSize)

	// Pares the multipart form
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		response := ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to parse multipart form: %v", err),
		}
		s.respondWithJSONError(w, response, http.StatusBadRequest)
		return
	}

	// Get the uploaded file
	file, fileHeader, err := r.FormFile("image")
	if err != nil {
		response := ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to get uploaded file: %v", err),
		}
		s.respondWithJSONError(w, response, http.StatusBadRequest)
		return
	}

	defer func(file multipart.File) {
		_ = file.Close()
	}(file)

	// validate the file
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	contentType := getImageContentType(ext)
	if contentType == "" {
		response := ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Unsupported file type: %s. Accepted types: .svg, .png, .jpg, .jpeg, .gif, .webp, .ico", ext),
		}
		s.respondWithJSONError(w, response, http.StatusBadRequest)
		return
	}

	// Read the file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		response := ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to read uploaded file: %v", err),
		}
		s.respondWithJSONError(w, response, http.StatusBadRequest)
		return
	}

	filename := s.generateImageFilename(fileContent, ext)

	uploadsDir := "images/uploads"
	if err := s.rootManager.MkdirAll(uploadsDir, 0755); err != nil {
		s.respondWithJSONError(w, ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to create uploads directory: %v", err),
		}, http.StatusInternalServerError)
		return
	}

	imagePath := filepath.Join(uploadsDir, filename)
	if err := s.rootManager.WriteFile(imagePath, fileContent, 0644); err != nil {
		s.respondWithJSONError(w, ImageUploadResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to save uploaded file: %v", err),
		}, http.StatusInternalServerError)
		return
	}

	imageURL := fmt.Sprintf("/images/uploads/%s", filename)
	response := ImageUploadResponse{
		Success: true,
		DataURI: imageURL,
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.showServerError(w, r, err)
	}
}

// generateImageFilename generates a unique filename for an image based on its content
func (s *Server) generateImageFilename(content []byte, ext string) string {
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])[:12] // Use first 12 characters of the hash
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%s-%s%s", timestamp, hashStr, ext)
}
