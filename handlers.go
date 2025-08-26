package main

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}

	content, err := s.dirManager.ReadFile(file.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// Check for message in query params (after redirect from save/daily)
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Message = msg
		data.MessageType = r.URL.Query().Get("type")
		if data.MessageType == "" {
			data.MessageType = "success"
		}
	}

	if err := s.executePage(w, "view.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRefreshCache(w http.ResponseWriter, r *http.Request) {
	s.refreshResourceCache()
	http.Redirect(w, r, "/resources", http.StatusSeeOther)
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) {
		http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
		return
	}

	content, err := s.dirManager.ReadFile(file.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")
	if err := s.dirManager.WriteString(file.Name, content); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/"+file.ID+"?msg=File saved successfully&type=success", http.StatusSeeOther)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// Check for message in query params (after redirect from add/delete)
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Message = msg
		data.MessageType = r.URL.Query().Get("type")
		if data.MessageType == "" {
			data.MessageType = "success"
		}
	}

	if err := s.executePage(w, "resources.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleCreateResource(w http.ResponseWriter, r *http.Request) {
	fileName := strings.TrimSpace(r.FormValue("filename"))
	if fileName == "" {
		http.Redirect(w, r, "/resources?msg=Filename cannot be empty&type=danger", http.StatusSeeOther)
		return
	}

	// Validate filename contains only allowed characters
	if !isValidFileName(fileName) {
		http.Redirect(w, r, "/resources?msg=Filename must contain only letters, numbers, dashes, periods, underscores, and forward slashes&type=danger", http.StatusSeeOther)
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
			http.Redirect(w, r, "/resources?msg=Failed to create directories&type=danger", http.StatusSeeOther)
			return
		}
	}

	// Check if file already exists
	if s.dirManager.Exists(fullPath) {
		http.Redirect(w, r, "/resources?msg=File already exists&type=danger", http.StatusSeeOther)
		return
	}

	// Create the new file with default content
	defaultContent := fmt.Sprintf("# %s\n\nCreated on %s\n\n",
		strings.TrimSuffix(filepath.Base(fileName), ".md"),
		time.Now().Format("2006-01-02"))

	if err := s.dirManager.WriteString(fullPath, defaultContent); err != nil {
		http.Redirect(w, r, "/resources?msg=Failed to create file&type=danger", http.StatusSeeOther)
		return
	}

	// Refresh the resource cache
	s.refreshResourceCache()

	// Redirect to the new file
	fileID := "resources_" + s.createID(fileName)
	http.Redirect(w, r, "/"+fileID+"?msg=File created successfully&type=success", http.StatusSeeOther)
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

func (s *Server) processInlineSVG(htmlContent string) string {
	// Replace <img> tags with inline SVG content
	re := regexp.MustCompile(`<img[^>]+src="([^">]+\.svg)"[^>]*>`)

	return re.ReplaceAllStringFunc(htmlContent, func(imgTag string) string {
		// Extract the icon path
		srcMatch := regexp.MustCompile(`src="([^">]+\.svg)"`).FindStringSubmatch(imgTag)
		if len(srcMatch) < 2 {
			return imgTag // No src found, return original tag
		}

		iconPath := strings.TrimPrefix(srcMatch[1], "/images/")
		svgContent := s.getInlineSVG(iconPath)
		if svgContent != "" {
			return svgContent
		}

		return imgTag // Return original tag if SVG not found
	})
}

func (s *Server) getInlineSVG(iconPath string) string {
	// Try user's path first
	userSVGPath := filepath.Join("images", iconPath)
	if s.dirManager.Exists(userSVGPath) {
		content, err := s.dirManager.ReadFile(userSVGPath)
		if err == nil {
			return string(content)
		}
	}

	// Fallback to static embedded files
	staticPath := "static/images/" + iconPath
	if file, err := staticFS.Open(staticPath); err == nil {
		defer func(file fs.File) {
			_ = file.Close()
		}(file)

		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			content, err := io.ReadAll(file)
			if err == nil {
				return string(content)
			}
		}
	}

	return ""
}
