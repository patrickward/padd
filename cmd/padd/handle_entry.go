package main

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/patrickward/padd"
)

// EntryConfig defines how to handle adding entries to a file
type EntryConfig struct {
	FileID         string
	RedirectPath   string
	EntryFormatter func(entry string, timestamp time.Time) string
	SectionConfig  *padd.SectionInsertionConfig // nil means use date insertion logic
}

func (s *Server) handleAddTemporalEntry(directory string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		entry := strings.TrimSpace(r.FormValue("entry"))

		// Make sure directory is one of the temporal directories
		if !slices.Contains(s.fileRepo.Config().TemporalDirectories(), directory) {
			s.flashManager.SetError(w, "Invalid directory")
			s.redirectTo(w, r, "/")
			return
		}

		if entry == "" {
			s.flashManager.SetError(w, "Entry cannot be empty")
			s.redirectTo(w, r, "/"+directory)
			return
		}

		doc, err := s.fileRepo.GetOrCreateTemporalDocument(directory, time.Now())
		if err != nil {
			s.flashManager.SetError(w, fmt.Sprintf("Failed to get daily document: %v", err))
			s.redirectTo(w, r, "/"+directory)
			return
		}

		config := padd.EntryInsertionConfig{
			Strategy:       padd.InsertByTimestamp,
			EntryFormatter: padd.TimestampEntryFormatter,
		}

		if err := doc.AddEntry(entry, config); err != nil {
			s.flashManager.SetError(w, fmt.Sprintf("Failed to add entry: %v", err))
			s.redirectTo(w, r, "/"+directory)
			return
		}

		s.flashManager.SetSuccess(w, "Entry added successfully")
		s.redirectTo(w, r, "/"+directory)
	}
}

// handleAddEntry handles adding entries to a specified file under a specific section
//
// Use the "section_header" form field to specify the section header (without ##). If not provided, defaults to prepending to file.
// Use the "as_task" form field to indicate if entry should be formatted as a task. Default is a simple list item.
func (s *Server) handleAddEntry(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	if fileID == "" {
		s.flashManager.SetError(w, "File ID is required")
		s.redirectTo(w, r, "/")
		return
	}

	// Look for a header field from the form.
	var header string
	if h := r.FormValue("section_header"); h != "" {
		header = "## " + strings.TrimSpace(h)
	}

	// Determine entry formatter based on "as_task" form field
	var entryFormatter func(string, time.Time) string
	if r.FormValue("as_task") == "true" {
		entryFormatter = padd.TaskEntryFormatter
	} else {
		entryFormatter = padd.NoteEntryFormatter
	}

	redirectPath := r.Referer()

	config := EntryConfig{
		FileID:         fileID,
		RedirectPath:   redirectPath,
		EntryFormatter: entryFormatter,
		SectionConfig: &padd.SectionInsertionConfig{
			SectionHeader:  header,
			InsertAtTop:    true,
			BlankLineAfter: false,
		},
	}

	s.addEntry(w, r, config)
}

// addEntry is a generic handler for adding entries to markdown files
func (s *Server) addEntry(w http.ResponseWriter, r *http.Request, config EntryConfig) {
	if r.Method != http.MethodPost {
		s.redirectTo(w, r, config.RedirectPath)
		return
	}

	entry := strings.TrimSpace(r.FormValue("entry"))
	if entry == "" {
		s.flashManager.SetError(w, "Entry cannot be empty")
		s.redirectTo(w, r, config.RedirectPath)
		return
	}

	doc, err := s.fileRepo.GetDocument(config.FileID)
	if err != nil {
		s.flashManager.SetError(w, "Invalid file ID")
		s.redirectTo(w, r, "/")
		return
	}

	insertionConfig := padd.EntryInsertionConfig{
		EntryFormatter: config.EntryFormatter,
	}

	if config.SectionConfig != nil {
		insertionConfig.Strategy = padd.InsertInSection
		insertionConfig.SectionConfig = config.SectionConfig
	} else {
		insertionConfig.Strategy = padd.InsertByTimestamp
	}

	if err := doc.AddEntry(entry, insertionConfig); err != nil {
		s.flashManager.SetError(w, fmt.Sprintf("Failed to add entry: %v", err))
		s.redirectTo(w, r, config.RedirectPath)
		return
	}

	s.flashManager.SetSuccess(w, "Entry added successfully")
	s.redirectTo(w, r, config.RedirectPath)
}
