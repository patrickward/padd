package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/patrickward/padd/internal/web"
)

// handleResources shows a list of available resource files
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	tree := s.fileRepo.DirectoryTreeFor(s.fileRepo.Config().ResourcesDirectory)

	data := web.PageData{
		Title:         "Resources",
		NavMenuFiles:  s.navigationMenu(r.URL.Path),
		IsResources:   true,
		DirectoryTree: tree,
	}

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}

	if err := s.executePage(w, "resources.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) handleCreateResource(w http.ResponseWriter, r *http.Request) {
	fileName := strings.TrimSpace(r.FormValue("filename"))
	if fileName == "" {
		s.flashManager.SetError(w, "Filename cannot be empty")
		s.redirectTo(w, r, "/resources")
		return
	}

	// Validate filename contains only allowed characters
	if !filenameIsValid(fileName) {
		s.flashManager.SetError(w, "Filename must contain only letters, numbers, dashes, periods, underscores, and forward slashes")
		s.redirectTo(w, r, "/resources")
		return
	}

	// Add .md extension if not present
	if !strings.HasSuffix(fileName, ".md") {
		fileName = fileName + ".md"
	}

	// Construct full path within resources directory
	fullPath := filepath.Join(resourcesDir, fileName)

	// Create directories if the filename contains path separators
	if strings.Contains(fileName, "/") {
		dir := filepath.Dir(fullPath)
		if err := s.rootManager.MkdirAll(dir, 0755); err != nil {
			s.flashManager.SetError(w, "Failed to create directories")
			s.redirectTo(w, r, "/resources")
			return
		}
	}

	// Check if file already exists
	if s.rootManager.FileExists(fullPath) {
		s.flashManager.SetError(w, "File already exists")
		s.redirectTo(w, r, "/resources")
		return
	}

	// Create the new file with default content
	defaultContent := fmt.Sprintf("---\ncreated_at: %s\n---\n",
		time.Now().Format("2006-01-02 15:04:05"))

	if err := s.rootManager.WriteString(fullPath, defaultContent); err != nil {
		s.flashManager.SetError(w, "Failed to create file")
		s.redirectTo(w, r, "/resources")
		return
	}

	// Refresh the resource cache
	s.fileRepo.ReloadResources()

	// Redirect to the new file
	fileID := "resources/" + s.fileRepo.CreateID(fileName)
	s.flashManager.SetSuccess(w, "File created successfully")
	s.redirectTo(w, r, "/"+fileID)
}

// handleRefreshResources refreshes the resource file cache and redirects back to the resources page
func (s *Server) handleRefreshResources(w http.ResponseWriter, r *http.Request) {
	s.fileRepo.ReloadResources()
	s.redirectTo(w, r, "/resources")
}

// filenameIsValid checks if a filename contains only allowed characters
func filenameIsValid(fileName string) bool {
	for _, r := range fileName {
		if !((r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '.' || r == '_' || r == '/') {
			return false
		}
	}
	return true
}

// handleResourcesAPI serves a JSON list of available resource files in their hierarchical structure
func (s *Server) handleResourcesAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tree := s.fileRepo.DirectoryTreeFor(s.fileRepo.Config().ResourcesDirectory)

	if err := json.NewEncoder(w).Encode(tree); err != nil {
		s.showServerError(w, nil, err)
	}
}
