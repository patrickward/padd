package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Server) showPageNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := s.executePage(w, "404.html", PageData{
		Title:     "Page Not Found - " + appName,
		CoreFiles: s.getCoreFiles(""),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) showServerError(w http.ResponseWriter, _ *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := s.executePage(w, "500.html", PageData{
		Title:        "Server Error - " + appName,
		CoreFiles:    s.getCoreFiles(""),
		ErrorMessage: err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	file, err := s.getFileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Name) {
		s.showServerError(w, r, fmt.Errorf("invalid file"))
		return
	}

	content, err := s.dirManager.ReadFile(file.Name)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	// Get the search query and match parameters
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	var searchMatch int
	if matchStr := r.URL.Query().Get("match"); matchStr != "" {
		_, _ = fmt.Sscanf(matchStr, "%d", &searchMatch)
	}

	// Render content with search highlighting if needed
	var renderedContent template.HTML
	if searchQuery != "" {
		renderedContent = s.renderMarkdownWithHighlight(string(content), searchQuery, searchMatch)
	} else {
		renderedContent = s.renderMarkdown(string(content))
	}

	data := PageData{
		Title:         file.Display + " - " + appName,
		CurrentFile:   file,
		Content:       renderedContent,
		RawContent:    string(content),
		CoreFiles:     s.getCoreFiles(file.Name),
		ResourceFiles: s.getResourceFiles(file.Name),
		SearchQuery:   searchQuery,
		SearchMatch:   searchMatch,
	}

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}

	if err := s.executePage(w, "view.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) handleRefreshCache(w http.ResponseWriter, r *http.Request) {
	s.refreshResourceCache()
	http.Redirect(w, r, "/resources", http.StatusSeeOther)
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	file, err := s.getFileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Name) {
		http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
		return
	}

	content, err := s.dirManager.ReadFile(file.Name)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	data := PageData{
		Title:         "Edit - " + file.Display + " - " + appName,
		CurrentFile:   file,
		RawContent:    string(content),
		IsEditing:     true,
		CoreFiles:     s.getCoreFiles(file.Name),
		ResourceFiles: s.getResourceFiles(file.Name),
	}

	if err := s.executePage(w, "edit.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	file, err := s.getFileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Name) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")
	if err := s.dirManager.WriteString(file.Name, content); err != nil {
		s.showServerError(w, r, err)
		return
	}

	s.flashManager.SetSuccess(w, "File saved successfully")
	http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
}

func (s *Server) handleDaily(w http.ResponseWriter, r *http.Request) {
	config := EntryConfig{
		FileName:       "daily.md",
		RedirectPath:   "/daily",
		EntryFormatter: s.dailyEntryFormatter,
		SectionConfig:  nil, // Use legacy daily insertion logic
	}

	s.handleAddEntry(w, r, config)
}

func (s *Server) handleInboxAdd(w http.ResponseWriter, r *http.Request) {
	var header string
	// Look for a header field from the form. If not found, use "## Quick Capture"
	if h := r.FormValue("section_header"); h != "" {
		header = "## " + strings.TrimSpace(h)
	} else {
		header = "## Quick Capture"
	}

	config := EntryConfig{
		FileName:       "inbox.md",
		RedirectPath:   "/inbox",
		EntryFormatter: s.inboxEntryFormatter,
		SectionConfig: &SectionInsertionConfig{
			SectionHeader:   header,
			CreateIfMissing: true,
			InsertAtTop:     true,
			BlankLineAfter:  false,
		},
	}

	s.handleAddEntry(w, r, config)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	results := make(map[string][]SearchMatch)

	// Search core files
	for _, file := range filesMap {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	// Search resource files
	resourceFiles := s.getResourceFiles("")
	for _, file := range resourceFiles {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	data := PageData{
		Title:         "Search Results - " + appName,
		IsSearching:   true,
		SearchQuery:   query,
		SearchResults: results,
		CoreFiles:     s.getCoreFiles(""),
		ResourceFiles: s.getResourceFiles(""),
	}

	if err := s.executePage(w, "search.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

// handleResources shows a list of available resource files
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	resourceFiles := s.getResourceFiles("")
	resourceTree := s.buildDirectoryTree(resourceFiles)

	data := PageData{
		Title:         "Resources - " + appName,
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
	if !isValidFileName(fileName) {
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
		if err := s.dirManager.MkdirAll(dir, 0755); err != nil {
			s.flashManager.SetError(w, "Failed to create directories")
			http.Redirect(w, r, "/resources", http.StatusSeeOther)
			return
		}
	}

	// Check if file already exists
	if s.dirManager.Exists(fullPath) {
		s.flashManager.SetError(w, "File already exists")
		http.Redirect(w, r, "/resources", http.StatusSeeOther)
		return
	}

	// Create the new file with default content
	defaultContent := fmt.Sprintf("# %s\n\nCreated on %s\n\n",
		strings.TrimSuffix(filepath.Base(fileName), ".md"),
		time.Now().Format("2006-01-02"))

	if err := s.dirManager.WriteString(fullPath, defaultContent); err != nil {
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

// isValidFileName checks if a filename contains only allowed characters
func isValidFileName(fileName string) bool {
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
		if s.dirManager.Exists(userImagePath) {
			content, err := s.dirManager.ReadFile(userImagePath)
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
		if file, err := staticFS.Open(staticPath); err == nil {
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
	_ = s.dirManager.WalkDir(userIconsDir, func(path string, d fs.DirEntry, err error) error {
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
	if entries, err := staticFS.ReadDir("static/images/icons"); err == nil {
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

// handleResourcesAPI serves a JSON list of available resource files in their hierarchical structure
func (s *Server) handleResourcesAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resourceFiles := s.getResourceFiles("")
	resourceTree := s.buildDirectoryTree(resourceFiles)

	if err := json.NewEncoder(w).Encode(resourceTree); err != nil {
		s.showServerError(w, nil, err)
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
