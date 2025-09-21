package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/patrickward/padd/internal/rendering"
	"github.com/patrickward/padd/internal/web"
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

func (s *Server) handleTemporalRoot(path string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		doc, err := s.fileRepo.GetOrCreateTemporalDocument(path, time.Now())
		if err != nil {
			s.showServerError(w, r, fmt.Errorf("failed to get or create temporal document: %w", err))
			return
		}

		s.redirectTo(w, r, "/"+doc.Info.ID)
	}
}

func (s *Server) renderDirectoryView(w http.ResponseWriter, r *http.Request, data web.PageData) {
	if err := s.executePage(w, "directory_view.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) processPageView(w http.ResponseWriter, r *http.Request) (web.PageData, bool) {
	id := r.PathValue("id")
	if id == "" {
		id = "inbox"
	}

	doc, err := s.fileRepo.GetDocument(id)
	if err != nil {
		s.showPageNotFound(w, r)
		return web.PageData{}, true
	}

	if doc.Info.IsDirectory {
		s.renderDirectoryView(w, r, web.PageData{
			Title:        doc.Info.TitleBase,
			CurrentFile:  doc.Info,
			NavMenuFiles: s.navigationMenu(doc.Info.ID),
		})
		return web.PageData{}, true
	}

	content, err := doc.Content()
	if err != nil {
		s.showServerError(w, r, fmt.Errorf("failed to get document content: %w", err))
		return web.PageData{}, true
	}

	// Get the search query and match parameters
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	var searchMatch int
	if matchStr := r.URL.Query().Get("match"); matchStr != "" {
		_, _ = fmt.Sscanf(matchStr, "%d", &searchMatch)
	}

	// Render content with search highlighting if needed
	var renderedContent rendering.RenderedContent
	if searchQuery != "" {
		renderedContent = s.renderer.RenderWithHighlight(string(content), searchQuery, searchMatch)
	} else {
		//renderedContent = s.renderMarkdown(string(content))
		renderedContent = s.renderer.Render(string(content))
	}

	if renderedContent.Title == "" {
		renderedContent.Title = doc.Info.TitleBase
	}

	data := web.PageData{
		Title:          renderedContent.Title,
		SectionHeaders: renderedContent.SectionHeaders,
		TasksTotal:     renderedContent.TasksTotal,
		TasksCompleted: renderedContent.TasksCompleted,
		TasksPending:   renderedContent.TasksPending,
		CurrentFile:    doc.Info,
		Content:        renderedContent.HTML,
		RawContent:     string(content),
		NavMenuFiles:   s.navigationMenu(doc.Info.ID),
		SearchQuery:    searchQuery,
		SearchMatch:    searchMatch,
	}

	data = s.addMetadataToPageData(data, renderedContent.Metadata)

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}
	return data, false
}
