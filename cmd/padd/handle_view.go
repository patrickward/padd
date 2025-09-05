package main

import (
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) handlePageHeader(w http.ResponseWriter, r *http.Request) {
	data, done := s.processPageView(w, r)
	if done {
		return
	}

	if err := s.executePage(w, "page_header.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	data, done := s.processPageView(w, r)
	if done {
		return
	}

	if err := s.executePage(w, "view.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) processPageView(w http.ResponseWriter, r *http.Request) (PageData, bool) {
	id := r.PathValue("id")

	// For daily/journal without specific month, redirect to current month
	if id == "daily" || id == "journal" {
		currentFile, err := s.getCurrentTemporalFile(id)
		if err != nil {
			s.showServerError(w, r, err)
			return PageData{}, true
		}

		http.Redirect(w, r, "/"+currentFile.ID, http.StatusSeeOther)
		return PageData{}, true
	}

	file, err := s.getFileInfo(id)
	if err != nil {
		s.showPageNotFound(w, r)
		return PageData{}, true
	}

	if !s.isValidFile(file.Path) {
		s.showServerError(w, r, fmt.Errorf("invalid file"))
		return PageData{}, true
	}

	content, err := s.dirManager.ReadFile(file.Path)
	if err != nil {
		s.showServerError(w, r, err)
		return PageData{}, true
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
		Title:             renderedContent.Title,
		SectionHeaders:    renderedContent.SectionHeaders,
		TasksCount:        renderedContent.TasksCount,
		HasCompletedTasks: renderedContent.HasCompletedTasks,
		CurrentFile:       file,
		Content:           renderedContent.HTML,
		RawContent:        string(content),
		CoreFiles:         s.getCoreFiles(file.Path),
		ResourceFiles:     s.getResourceFiles(file.Path),
		SearchQuery:       searchQuery,
		SearchMatch:       searchMatch,
	}

	data = s.addMetadataToPageData(data, renderedContent.Metadata)

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}
	return data, false
}
