package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// SectionInsertionConfig defines how to insert entries under specific headers
type SectionInsertionConfig struct {
	SectionHeader  string // The ## header to look for (e.g., "## Inbox Dump")
	InsertAtTop    bool   // true = insert at top of section, false = at bottom
	BlankLineAfter bool   // Add blank line after new entry
}

// EntryConfig defines how to handle adding entries to a file
type EntryConfig struct {
	FileID         string
	RedirectPath   string
	EntryFormatter func(entry string, timestamp time.Time) string
	SectionConfig  *SectionInsertionConfig // nil means use date insertion logic
}

// handleDailyEntry handles adding entries to a month-based daily file under the daily/<year>/<monthly> hierarchy
func (s *Server) handleDailyEntry(w http.ResponseWriter, r *http.Request) {
	entry := strings.TrimSpace(r.FormValue("entry"))
	err := s.addTemporalEntry(entry, "daily", time.Now())
	if err != nil {
		s.flashManager.SetError(w, fmt.Sprintf("Failed to add daily entry: %v", err))
		http.Redirect(w, r, "/daily", http.StatusSeeOther)
		return
	}

	s.flashManager.SetSuccess(w, "Daily entry added successfully")
	http.Redirect(w, r, "/daily", http.StatusSeeOther)
}

// handleJournalEntry handles adding entries to a month-based journal file under the journal/<year>/<monthly> hierarchy
func (s *Server) handleJournalEntry(w http.ResponseWriter, r *http.Request) {
	entry := strings.TrimSpace(r.FormValue("entry"))
	err := s.addTemporalEntry(entry, "journal", time.Now())
	if err != nil {
		s.flashManager.SetError(w, fmt.Sprintf("Failed to add journal entry: %v", err))
		http.Redirect(w, r, "/journal", http.StatusSeeOther)
		return
	}

	s.flashManager.SetSuccess(w, "Journal entry added successfully")
	http.Redirect(w, r, "/journal", http.StatusSeeOther)
}

func (s *Server) addTemporalEntry(entry, fileType string, timestamp time.Time) error {
	if entry == "" {
		return fmt.Errorf("entry cannot be empty")
	}

	filePath, err := s.rootManager.ResolveMonthlyFile(timestamp, fileType)
	if err != nil {
		return fmt.Errorf("failed to resolve %s file: %v", fileType, err)
	}

	existingContent, err := s.rootManager.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read %s file: %v", fileType, err)
	}

	formattedEntry := s.timestampEntryFormatter(entry, timestamp)
	lines := strings.Split(string(existingContent), "\n")
	result := s.insertTimestampEntry(lines, formattedEntry, timestamp)
	updatedContent := strings.Join(result, "\n")

	if err := s.rootManager.WriteString(filePath, updatedContent); err != nil {
		return fmt.Errorf("failed to write to %s file: %v", fileType, err)
	}

	return nil
}

// handleAddEntry handles adding entries to a specified file under a specific section
//
// Use the "section_header" form field to specify the section header (without ##). If not provided, defaults to prepending to file.
// Use the "as_task" form field to indicate if entry should be formatted as a task. Default is a simple list item.
func (s *Server) handleAddEntry(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	if fileID == "" {
		s.flashManager.SetError(w, "File ID is required")
		http.Redirect(w, r, "/", http.StatusSeeOther)
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
		entryFormatter = s.taskEntryFormatter
	} else {
		//entryFormatter = s.listEntryFormatter
		entryFormatter = s.noteEntryFormatter
	}

	redirectPath := r.Referer()

	config := EntryConfig{
		FileID:         fileID,
		RedirectPath:   redirectPath,
		EntryFormatter: entryFormatter,
		SectionConfig: &SectionInsertionConfig{
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
		http.Redirect(w, r, config.RedirectPath, http.StatusSeeOther)
		return
	}

	entry := strings.TrimSpace(r.FormValue("entry"))
	if entry == "" {
		s.flashManager.SetError(w, "Entry cannot be empty")
		http.Redirect(w, r, config.RedirectPath, http.StatusSeeOther)
		return
	}

	// Get the file info
	fileInfo, err := s.fileRepo.FileInfo(config.FileID)
	if err != nil {
		s.flashManager.SetError(w, "Invalid file ID")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Read existing content
	existingContent, err := s.rootManager.ReadFile(fileInfo.Path)
	if err != nil {
		s.flashManager.SetError(w, "Failed to read file")
		http.Redirect(w, r, config.RedirectPath, http.StatusSeeOther)
		return
	}

	now := time.Now()
	formattedEntry := config.EntryFormatter(entry, now)

	lines := strings.Split(string(existingContent), "\n")

	var result []string
	if config.SectionConfig != nil {
		result = s.insertIntoSection(lines, formattedEntry, *config.SectionConfig)
	} else {
		// Fallback to date insertion (for daily/journal entries)
		result = s.insertTimestampEntry(lines, formattedEntry, now)
	}

	updatedContent := strings.Join(result, "\n")
	if err := s.rootManager.WriteString(fileInfo.Path, updatedContent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.flashManager.SetSuccess(w, "Entry added successfully")
	http.Redirect(w, r, config.RedirectPath, http.StatusSeeOther)
}

// insertIntoSection handles insertion under specific ## headers
func (s *Server) insertIntoSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
	// Find the target section (normalize whitespace for comparison)
	sectionStartIdx := -1
	sectionEndIdx := len(lines)
	targetHeader := strings.TrimSpace(config.SectionHeader)

	// If the target header is empty, we treat it as not found and just prepend to top
	if targetHeader == "##" || targetHeader == "" {
		log.Println("No valid section header provided; prepending entry to top of file")
		return append([]string{formattedEntry}, lines...)
	}

	for i, line := range lines {
		normalizedLine := strings.TrimSpace(line)
		if normalizedLine == targetHeader {
			sectionStartIdx = i
			// Find the end of this section (next ## header or end of file)
			for j := i + 1; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "## ") {
					sectionEndIdx = j
					break
				}
			}
			break
		}
	}

	// If the section not found, create it at the top
	if sectionStartIdx == -1 {
		log.Println("Section not found; creating new section: ", config.SectionHeader)
		return s.createNewSection(lines, formattedEntry, config)
	}

	// Insert the entry at the specified position within the section
	result := make([]string, 0, len(lines)+2)

	if config.InsertAtTop {
		// Insert right after the section header
		insertPos := sectionStartIdx + 1
		// Skip any blank lines immediately after header
		for insertPos < sectionEndIdx && strings.TrimSpace(lines[insertPos]) == "" {
			insertPos++
		}

		result = append(result, lines[:insertPos]...)
		result = append(result, formattedEntry)
		if config.BlankLineAfter {
			result = append(result, "")
		}
		result = append(result, lines[insertPos:]...)
	} else {
		// Insert at bottom of section (before next ## header)
		result = append(result, lines[:sectionEndIdx]...)
		result = append(result, formattedEntry)
		if config.BlankLineAfter {
			result = append(result, "")
		}
		result = append(result, lines[sectionEndIdx:]...)
	}

	return result
}

// createNewSection creates a new section and adds the entry
func (s *Server) createNewSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
	// Find where to insert the new section (after main header)
	insertPos := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			insertPos = i + 1
			// Skip blank lines after main header
			for insertPos < len(lines) && strings.TrimSpace(lines[insertPos]) == "" {
				insertPos++
			}
			break
		}
	}

	result := make([]string, 0, len(lines)+4)
	result = append(result, lines[:insertPos]...)
	result = append(result, config.SectionHeader)
	result = append(result, formattedEntry)
	if config.BlankLineAfter {
		result = append(result, "")
	}
	result = append(result, "") // Blank line after section
	if insertPos < len(lines) {
		result = append(result, lines[insertPos:]...)
	}

	return result
}

// insertTimestampEntry handles the hierarchical date insertion logic
func (s *Server) insertTimestampEntry(lines []string, formattedEntry string, timestamp time.Time) []string {
	dayHeader := fmt.Sprintf("## %s", timestamp.Format("Monday, January 2, 2006"))

	// Find insertion point after the main header
	insertPos := 0
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			insertPos = i + 1
			// Skip blank lines after main header
			for insertPos < len(lines) && strings.TrimSpace(lines[insertPos]) == "" {
				insertPos++
			}
			break
		}
	}

	// Now, look for an existing day header starting from the insertion point
	for i := insertPos; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// If we find our exact day header
		if line == dayHeader {
			// Add entry right after it
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:i+1]...)
			result = append(result, formattedEntry)
			result = append(result, lines[i+1:]...)
			return result
		}
	}

	// Day header doesn't exist - add it at the top
	result := make([]string, 0, len(lines)+3)
	result = append(result, lines[:insertPos]...)
	result = append(result, dayHeader)
	result = append(result, formattedEntry)
	result = append(result, "")
	result = append(result, lines[insertPos:]...)
	return result
}

// Entry formatters
func (s *Server) noteEntryFormatter(entry string, _ time.Time) string {
	return fmt.Sprintf("%s\n", entry)
}

func (s *Server) taskEntryFormatter(entry string, _ time.Time) string {
	return fmt.Sprintf("- [ ] %s", entry)
}

func (s *Server) timestampEntryFormatter(entry string, timestamp time.Time) string {
	return fmt.Sprintf("### %s\n\n%s\n", timestamp.Format("03:04:05 PM"), entry)
}
