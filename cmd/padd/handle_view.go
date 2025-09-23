package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/patrickward/padd/internal/files"
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

	// Check if this is a CSV file
	if strings.HasSuffix(strings.ToLower(doc.Info.Path), ".csv") {
		return s.renderCsvView(w, r, doc)
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

func (s *Server) renderCsvView(w http.ResponseWriter, r *http.Request, doc *files.Document) (web.PageData, bool) {
	csvDoc := files.NewCSVDocument(doc)

	// Get CSV records
	records, err := csvDoc.GetRecords()
	if err != nil {
		s.showDocumentError(w, r, csvDoc.Document, fmt.Errorf("failed to get CSV records: %w", err))
		return web.PageData{}, true
	}

	// Get CSV metadata
	metadata, err := csvDoc.GetMetadata()
	if err != nil {
		s.showServerError(w, r, fmt.Errorf("failed to get CSV metadata: %w", err))
		return web.PageData{}, true
	}

	// Apply sorting if specified in metadata
	if metadata.SortColumn != "" && len(records) > 1 {
		records = s.sortCSVRecords(records, metadata)
	}

	// Determine the title
	title := metadata.Title
	if title == "" {
		title = doc.Info.TitleBase
	}

	data := web.PageData{
		Title:        title,
		Description:  metadata.Description,
		CurrentFile:  doc.Info,
		NavMenuFiles: s.navigationMenu(doc.Info.ID),
	}

	rowCount, _ := csvDoc.RecordCount()
	colCount, _ := csvDoc.ColumnCount()

	csvData := &web.CSVData{
		Records:     records,
		Metadata:    metadata,
		RecordCount: rowCount,
		ColumnCount: colCount,
	}

	data.CSVData = csvData

	// Check for a flash message
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}

	if err := s.executePage(w, "view_csv.html", data); err != nil {
		s.showServerError(w, r, err)
	}

	return data, true
}

// sortCSVRecords sorts CSV records based on metadata sort settings
func (s *Server) sortCSVRecords(records [][]string, metadata *files.CSVMetadata) [][]string {
	if len(records) <= 1 {
		return records
	}

	// Find the column index to sort by
	sortColIndex := -1

	// Try to match by column name first (if headers are defined)
	if len(metadata.Headers) > 0 && len(records) > 0 {
		for i, header := range metadata.Headers {
			if header == metadata.SortColumn {
				sortColIndex = i
				break
			}
		}
	}

	// If not found by name, try to match against the actual first row
	if sortColIndex == -1 && len(records[0]) > 0 {
		for i, header := range records[0] {
			if header == metadata.SortColumn {
				sortColIndex = i
				break
			}
		}
	}

	// If still not found, return unsorted
	if sortColIndex == -1 || sortColIndex >= len(records[0]) {
		return records
	}

	// Create a copy to sort (preserve header row)
	header := records[0]
	dataRows := make([][]string, len(records)-1)
	copy(dataRows, records[1:])

	// Simple sort implementation
	for i := 0; i < len(dataRows)-1; i++ {
		for j := 0; j < len(dataRows)-i-1; j++ {
			if len(dataRows[j]) <= sortColIndex || len(dataRows[j+1]) <= sortColIndex {
				continue
			}

			val1 := strings.ToLower(strings.TrimSpace(dataRows[j][sortColIndex]))
			val2 := strings.ToLower(strings.TrimSpace(dataRows[j+1][sortColIndex]))

			shouldSwap := false
			if metadata.SortDesc {
				shouldSwap = val1 < val2
			} else {
				shouldSwap = val1 > val2
			}

			if shouldSwap {
				dataRows[j], dataRows[j+1] = dataRows[j+1], dataRows[j]
			}
		}
	}

	// Reconstruct with header
	result := make([][]string, len(records))
	result[0] = header
	copy(result[1:], dataRows)

	return result
}
