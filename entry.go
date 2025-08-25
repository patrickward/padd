package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// SectionInsertionConfig defines how to insert entries under specific headers
type SectionInsertionConfig struct {
	SectionHeader   string // The ## header to look for (e.g., "## Inbox Dump")
	CreateIfMissing bool   // Whether to create the section if it doesn't exist
	InsertAtTop     bool   // true = insert at top of section, false = at bottom
	BlankLineAfter  bool   // Add blank line after new entry
}

// EntryConfig defines how to handle adding entries to a file
type EntryConfig struct {
	FileName       string
	RedirectPath   string
	EntryFormatter func(entry string, timestamp time.Time) string
	SectionConfig  *SectionInsertionConfig // nil means use date insertion logic
}

// handleAddEntry is a generic handler for adding entries to markdown files
func (s *Server) handleAddEntry(w http.ResponseWriter, r *http.Request, config EntryConfig) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, config.RedirectPath, http.StatusSeeOther)
		return
	}

	entry := strings.TrimSpace(r.FormValue("entry"))
	if entry == "" {
		http.Redirect(w, r, config.RedirectPath+"?msg=Entry cannot be empty&type=danger", http.StatusSeeOther)
		return
	}

	// Read existing content
	existingContent, err := s.dirManager.ReadFile(config.FileName)
	if err != nil {
		// Create basic content if file doesn't exist
		title := strings.Title(strings.TrimSuffix(config.FileName, ".md"))
		existingContent = []byte(fmt.Sprintf("# %s\n\n", title))
	}

	now := time.Now()
	formattedEntry := config.EntryFormatter(entry, now)

	lines := strings.Split(string(existingContent), "\n")

	var result []string
	if config.SectionConfig != nil {
		result = s.insertIntoSection(lines, formattedEntry, *config.SectionConfig)
	} else {
		// Fallback to date insertion (for daily entries)
		result = s.insertDaily(lines, entry, now)
	}

	updatedContent := strings.Join(result, "\n")
	if err := s.dirManager.WriteString(config.FileName, updatedContent); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := fmt.Sprintf("Entry added at %s", now.Format("15:04:05"))
	http.Redirect(w, r, config.RedirectPath+"?msg="+msg+"&type=success", http.StatusSeeOther)
}

// insertIntoSection handles insertion under specific ## headers
func (s *Server) insertIntoSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
	// Find the target section (normalize whitespace for comparison)
	sectionStartIdx := -1
	sectionEndIdx := len(lines)
	targetHeader := strings.TrimSpace(config.SectionHeader)

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

	// If section doesn't exist and we should create it
	if sectionStartIdx == -1 && config.CreateIfMissing {
		return s.createNewSection(lines, formattedEntry, config)
	}

	// If section doesn't exist and we shouldn't create it, just append at end
	if sectionStartIdx == -1 {
		result := make([]string, 0, len(lines)+2)
		result = append(result, lines...)
		result = append(result, formattedEntry)
		return result
	}

	// Insert into existing section
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

// insertDaily handles the date insertion logic
func (s *Server) insertDaily(lines []string, entry string, timestamp time.Time) []string {
	dateHeader := fmt.Sprintf("## %s", timestamp.Format("2006-01-02"))
	formattedEntry := fmt.Sprintf("- `%s` %s", timestamp.Format("15:04:05"), entry)

	var result []string
	dateFound := false

	for _, line := range lines {
		if line == dateHeader {
			dateFound = true
			result = append(result, line)
			result = append(result, formattedEntry)
		} else {
			result = append(result, line)
		}
	}

	// If date header wasn't found, add it at the top
	if !dateFound {
		insertPos := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				insertPos = i + 1
				for insertPos < len(lines) && strings.TrimSpace(lines[insertPos]) == "" {
					insertPos++
				}
				break
			}
		}

		result = nil
		result = append(result, lines[:insertPos]...)
		result = append(result, dateHeader)
		result = append(result, formattedEntry)
		result = append(result, "")
		if insertPos < len(lines) {
			result = append(result, lines[insertPos:]...)
		}
	}

	return result
}

// Entry formatters
func (s *Server) dailyEntryFormatter(entry string, timestamp time.Time) string {
	return fmt.Sprintf("- `%s` %s", timestamp.Format("15:04:05"), entry)
}

func (s *Server) inboxEntryFormatter(entry string, timestamp time.Time) string {
	return fmt.Sprintf("- %s", entry)
}
