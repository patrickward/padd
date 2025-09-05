package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// handleResources shows a list of available resource files
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	resourceFiles := s.getResourceFiles("")
	resourceTree := s.buildDirectoryTree(resourceFiles)

	data := PageData{
		Title:         "Resources",
		CoreFiles:     s.getCoreFiles(""),
		IsResources:   true,
		ResourceFiles: resourceFiles,
		ResourceTree:  resourceTree,
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
		http.Redirect(w, r, "/resources", http.StatusSeeOther)
		return
	}

	// Validate filename contains only allowed characters
	if !filenameIsValid(fileName) {
		s.flashManager.SetError(w, "Filename must contain only letters, numbers, dashes, periods, underscores, and forward slashes")
		http.Redirect(w, r, "/resources", http.StatusSeeOther)
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
			http.Redirect(w, r, "/resources", http.StatusSeeOther)
			return
		}
	}

	// Check if file already exists
	if s.rootManager.FileExists(fullPath) {
		s.flashManager.SetError(w, "File already exists")
		http.Redirect(w, r, "/resources", http.StatusSeeOther)
		return
	}

	// Create the new file with default content
	defaultContent := fmt.Sprintf("\n\n_Created on %s_\n\n",
		time.Now().Format("2006-01-02 15:04:05"))

	if err := s.rootManager.WriteString(fullPath, defaultContent); err != nil {
		s.flashManager.SetError(w, "Failed to create file")
		http.Redirect(w, r, "/resources", http.StatusSeeOther)
		return
	}

	// Refresh the resource cache
	s.refreshResourceCache()

	// Redirect to the new file
	fileID := "resources/" + s.createID(fileName)
	s.flashManager.SetSuccess(w, "File created successfully")
	http.Redirect(w, r, "/"+fileID, http.StatusSeeOther)
}

// handleRefreshResources refreshes the resource file cache and redirects back to the resources page
func (s *Server) handleRefreshResources(w http.ResponseWriter, r *http.Request) {
	s.refreshResourceCache()
	http.Redirect(w, r, "/resources", http.StatusSeeOther)
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

	resourceFiles := s.getResourceFiles("")
	resourceTree := s.buildDirectoryTree(resourceFiles)

	if err := json.NewEncoder(w).Encode(resourceTree); err != nil {
		s.showServerError(w, nil, err)
	}
}
