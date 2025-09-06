package padd

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// EntryInsertionStrategy defines how to insert entries into a file
type EntryInsertionStrategy int

const (
	// InsertInSection inserts the entry at the top of the section
	InsertInSection EntryInsertionStrategy = iota
	// InsertByTimestamp inserts the entry at the top of the section, sorted by timestamp
	InsertByTimestamp
	// PrependToFile inserts the entry at the top of the file
	PrependToFile
	// AppendToFile inserts the entry at the bottom of the file
	AppendToFile
)

// EntryInsertionConfig defines how to insert entries into a file
type EntryInsertionConfig struct {
	Strategy       EntryInsertionStrategy
	EntryFormatter func(entry string, timestamp time.Time) string
	SectionConfig  *SectionInsertionConfig
}

type SectionInsertionConfig struct {
	SectionHeader  string // The ## header to look for (e.g., "## Inbox Dump")
	InsertAtTop    bool   // true = insert at top of section, false = at bottom
	BlankLineAfter bool   // Add blank line after new entry
}

type Document struct {
	Info    FileInfo
	repo    *FileRepository
	content string
	loaded  bool
}

// Load reads the document from disk
func (d *Document) Load() error {
	if d.loaded {
		return nil
	}

	content, err := d.repo.rootManager.ReadFile(d.Info.Path)
	if err != nil {
		return fmt.Errorf("failed to load document %s: %w", d.Info.Path, err)
	}

	d.content = string(content)
	d.loaded = true
	return nil
}

// Content returns the content of the document
func (d *Document) Content() (string, error) {
	if err := d.Load(); err != nil {
		return "", err
	}

	return d.content, nil
}

// Save writes the document to disk
func (d *Document) Save(content string) error {
	if err := d.repo.rootManager.WriteString(d.Info.Path, content); err != nil {
		return fmt.Errorf("failed to save document %s: %w", d.Info.Path, err)
	}

	d.content = content
	d.loaded = true
	return nil
}

// Delete deletes the document from disk
func (d *Document) Delete() error {
	return d.repo.rootManager.Remove(d.Info.Path)
}

// AddEntry adds content to the document
func (d *Document) AddEntry(entry string, config EntryInsertionConfig) error {
	if err := d.Load(); err != nil {
		return err
	}

	if d.content == "" {
		d.content = entry
		return nil
	}

	lines := strings.Split(d.content, "\n")
	formattedEntry := config.EntryFormatter(entry, time.Now())

	var result []string
	switch config.Strategy {
	case InsertInSection:
		result = d.insertInSection(lines, formattedEntry, *config.SectionConfig)
	case InsertByTimestamp:
		result = d.insertByTimestamp(lines, formattedEntry, time.Now())
	case PrependToFile:
		result = append([]string{formattedEntry}, lines...)
	case AppendToFile:
		result = append(lines, formattedEntry)
	default:
		return fmt.Errorf("unsupported entry insertion strategy: %d", config.Strategy)
	}

	return d.Save(strings.Join(result, "\n"))
}

func (d *Document) insertInSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
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
		return d.createNewSection(lines, formattedEntry, config)
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

func (d *Document) insertByTimestamp(lines []string, formattedEntry string, timestamp time.Time) []string {
	dayHeader := fmt.Sprintf("## %s", timestamp.Format("Monday, January 2, 2006"))

	// Find the insertion point after the main header
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

// createNewSection creates a new section and adds the entry
func (d *Document) createNewSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
	// Find where to insert the new section (after main header)
	insertPos := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			insertPos = i + 1
			// Skip blank lines after the main header
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
