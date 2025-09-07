package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/patrickward/padd"
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

func (s *Server) processPageView(w http.ResponseWriter, r *http.Request) (padd.PageData, bool) {
	id := r.PathValue("id")
	if id == "" {
		id = "inbox"
	}

	if s.fileRepo.IsTemporalRoot(id) {
		doc, err := s.fileRepo.GetOrCreateTemporalDocument(id, time.Now())
		if err != nil {
			s.showServerError(w, r, err)
			return padd.PageData{}, true
		}

		http.Redirect(w, r, "/"+doc.Info.ID, http.StatusSeeOther)
		return padd.PageData{}, true
	}

	doc, err := s.fileRepo.GetDocument(id)
	if err != nil {
		s.showServerError(w, r, err)
		return padd.PageData{}, true
	}

	content, err := doc.Content()
	if err != nil {
		s.showServerError(w, r, err)
		return padd.PageData{}, true
	}

	// Get the search query and match parameters
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	var searchMatch int
	if matchStr := r.URL.Query().Get("match"); matchStr != "" {
		_, _ = fmt.Sscanf(matchStr, "%d", &searchMatch)
	}

	// Render content with search highlighting if needed
	var renderedContent padd.RenderedContent
	if searchQuery != "" {
		renderedContent = s.renderer.RenderWithHighlight(string(content), searchQuery, searchMatch)
	} else {
		//renderedContent = s.renderMarkdown(string(content))
		renderedContent = s.renderer.Render(string(content))
	}

	if renderedContent.Title == "" {
		renderedContent.Title = doc.Info.DisplayBase
	}

	data := padd.PageData{
		Title:             renderedContent.Title,
		SectionHeaders:    renderedContent.SectionHeaders,
		TasksCount:        renderedContent.TasksCount,
		HasCompletedTasks: renderedContent.HasCompletedTasks,
		CurrentFile:       doc.Info,
		Content:           renderedContent.HTML,
		RawContent:        string(content),
		NavMenuFiles:      s.navigationMenu(doc.Info.ID),
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
