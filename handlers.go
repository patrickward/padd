package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
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
		CanEdit:       file.Name != "daily.md",
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

	if !s.isValidFile(file.Name) || file.Name == "daily.md" {
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

	if !s.isValidFile(file.Name) || file.Name == "daily.md" {
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
	config := EntryConfig{
		FileName:       "inbox.md",
		RedirectPath:   "/inbox",
		EntryFormatter: s.inboxEntryFormatter,
		SectionConfig: &SectionInsertionConfig{
			SectionHeader:   "## Quick Capture",
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

	if err := s.executePage(w, "resources.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
