package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	// For daily/journal without specific month, redirect to current month
	if id == "daily" || id == "journal" {
		currentFile, err := s.getCurrentTemporalFile(id)
		if err != nil {
			s.showServerError(w, r, err)
			return
		}

		http.Redirect(w, r, "/"+currentFile.ID, http.StatusSeeOther)
		return
	}

	file, err := s.getFileInfo(id)
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Path) {
		s.showServerError(w, r, fmt.Errorf("invalid file"))
		return
	}

	content, err := s.dirManager.ReadFile(file.Path)
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
	var renderedContent RenderedContent
	if searchQuery != "" {
		renderedContent = s.renderMarkdownWithHighlight(string(content), searchQuery, searchMatch)
	} else {
		renderedContent = s.renderMarkdown(string(content))
	}

	if renderedContent.Title == "" {
		renderedContent.Title = file.DisplayBase
	}

	data := PageData{
		Title:          renderedContent.Title,
		SectionHeaders: renderedContent.SectionHeaders,
		HasTasks:       renderedContent.HasTasks,
		CurrentFile:    file,
		Content:        renderedContent.HTML,
		RawContent:     string(content),
		CoreFiles:      s.getCoreFiles(file.Path),
		ResourceFiles:  s.getResourceFiles(file.Path),
		SearchQuery:    searchQuery,
		SearchMatch:    searchMatch,
	}

	data = s.addMetadataToPageData(data, renderedContent.Metadata)

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}

	if err := s.executePage(w, "view.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}
