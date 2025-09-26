package files

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/patrickward/padd/internal/contentutil"
	"github.com/patrickward/padd/internal/crypto"
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
	EntryTimestamp time.Time
	EntryFormatter func(entry string, timestamp time.Time) string
	SectionConfig  *SectionInsertionConfig
}

func (c EntryInsertionConfig) Timestamp() time.Time {
	if c.EntryTimestamp.IsZero() {
		return time.Now()
	}
	return c.EntryTimestamp
}

type SectionInsertionConfig struct {
	SectionHeader  string // The ## header to look for (e.g., "## Inbox Dump")
	InsertAtTop    bool   // true = insert at top of section, false = at bottom
	BlankLineAfter bool   // Add blank line after new entry
}

type Document struct {
	Info           FileInfo
	repo           *FileRepository
	content        string
	loaded         bool
	taskCache      []Task
	taskCacheValid bool
	taskMu         sync.RWMutex
}

// load reads the document from disk
func (d *Document) load() error {
	if d.loaded {
		return nil
	}

	content, err := d.repo.rootManager.ReadFile(d.Info.Path)
	if err != nil {
		return fmt.Errorf("failed to load document %s: %w", d.Info.Path, err)
	}

	if d.repo.encryptionManager.IsActive() &&
		d.repo.encryptionManager.HasIdentities() {
		if crypto.IsAgeEncrypted(content) {
			decrypted, err := d.repo.encryptionManager.Decrypt(content)
			if err != nil {
				return fmt.Errorf("failed to decrypt document %s: %w", d.Info.Path, err)
			}
			d.content = decrypted
		} else {
			d.content = string(content)
		}
	} else {
		d.content = string(content)
	}

	d.loaded = true
	return nil
}

// Content returns the content of the document
func (d *Document) Content() (string, error) {
	if err := d.load(); err != nil {
		return "", err
	}

	return d.content, nil
}

// Save writes the document to disk
func (d *Document) Save(content string) error {
	// Remove space at the front of the content
	content = strings.TrimSpace(content)
	content += "\n"

	if d.repo.encryptionManager.IsActive() &&
		d.repo.encryptionManager.HasRecipients() &&
		crypto.HasEncryptedFrontmatter(content) {

		encrypted, err := d.repo.encryptionManager.Encrypt(content)
		if err != nil {
			return fmt.Errorf("failed to encrypt document %s: %w", d.Info.Path, err)
		}

		if err := d.repo.rootManager.WriteFile(d.Info.Path, encrypted, 0644); err != nil {
			return fmt.Errorf("failed to save document %s: %w", d.Info.Path, err)
		}
	} else {
		if err := d.repo.rootManager.WriteString(d.Info.Path, content); err != nil {
			return fmt.Errorf("failed to save document %s: %w", d.Info.Path, err)
		}
	}

	d.content = content
	d.loaded = true
	d.invalidateTaskCache()

	return nil
}

// Delete deletes the document from disk
func (d *Document) Delete() error {
	return d.repo.rootManager.Remove(d.Info.Path)
}

// AddEntry adds content to the document
func (d *Document) AddEntry(entry string, config EntryInsertionConfig) error {
	if err := d.load(); err != nil {
		return err
	}

	if d.content == "" {
		d.content = entry
		return nil
	}

	lines := contentutil.SplitLines(d.content)
	formattedEntry := config.EntryFormatter(entry, config.Timestamp())

	var result []string
	switch config.Strategy {
	case InsertInSection:
		result = d.insertInSection(lines, formattedEntry, *config.SectionConfig)
	case InsertByTimestamp:
		result = d.insertByTimestamp(lines, formattedEntry, config.Timestamp())
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
		// Need to find the frontmatter and insert the entry after it
		bounds := contentutil.FindFrontmatter(lines)
		if bounds.Found {
			insertPos := bounds.End + 1
			sectionStartIdx = insertPos
		} else {
			return append([]string{formattedEntry}, lines...)
		}
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

	// If the section is not found, create it at the top
	if sectionStartIdx == -1 {
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

	// Find the insertion point after any frontmatter and the main header
	insertPos := 0

	// Look for frontmatter and skip it
	bounds := contentutil.FindFrontmatter(lines)
	if bounds.Found {
		insertPos = bounds.End + 1
	}

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

	// Look for existing day headers and find the correct position
	var dateHeaders []struct {
		index int
		date  time.Time
	}

	for i := insertPos; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// If we find our exact day header, just add it and return
		if line == dayHeader {
			result := make([]string, 0, len(lines)+1)
			result = append(result, lines[:i+1]...)
			result = append(result, "")
			result = append(result, formattedEntry)
			result = append(result, lines[i+1:]...)
			return result
		}

		// Check if this is a day header
		if strings.HasPrefix(line, "## ") && len(line) > 3 {
			headerDateStr := line[3:]
			if headerDate, err := time.Parse("Monday, January 2, 2006", headerDateStr); err == nil {
				dateHeaders = append(dateHeaders, struct {
					index int
					date  time.Time
				}{i, headerDate})
			}
		}
	}

	// If no existing date headers found, insert at the top
	if len(dateHeaders) == 0 {
		result := make([]string, 0, len(lines)+3)
		result = append(result, lines[:insertPos]...)
		result = append(result, dayHeader)
		result = append(result, "")
		result = append(result, formattedEntry)
		result = append(result, "")
		result = append(result, lines[insertPos:]...)
		return result
	}

	// Find the correct position for the new entry
	// (should be in reverse chronological order - newest first)
	insertIdx := -1
	for _, header := range dateHeaders {
		if header.date.Before(timestamp) {
			insertIdx = header.index
			break
		}
	}

	result := make([]string, 0, len(lines)+2)

	if insertIdx == -1 {
		// Insertion date is newer than all existing dates - add to top
		result = append(result, lines...)
		result = append(result, "")
		result = append(result, dayHeader)
		result = append(result, "")
		result = append(result, formattedEntry)
	} else {
		// Insert at the specified position
		result = append(result, lines[:insertIdx]...)
		result = append(result, dayHeader)
		result = append(result, "")
		result = append(result, formattedEntry)
		result = append(result, "")
		result = append(result, lines[insertIdx:]...)
	}

	return result
}

// createNewSection creates a new section and adds the entry
func (d *Document) createNewSection(lines []string, formattedEntry string, config SectionInsertionConfig) []string {
	// Find the insertion point after any frontmatter and the main header
	insertPos := 0

	// Look for frontmatter and skip it
	bounds := contentutil.FindFrontmatter(lines)
	if bounds.Found {
		insertPos = bounds.End + 1
	}

	// Look for main header
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

// Entry formatters

func NoteEntryFormatter(entry string, _ time.Time) string {
	return fmt.Sprintf("%s\n", entry)
}

func TaskEntryFormatter(entry string, _ time.Time) string {
	return fmt.Sprintf("- [ ] %s", entry)
}

func TimestampEntryFormatter(entry string, timestamp time.Time) string {
	return fmt.Sprintf("### %s\n\n%s\n", timestamp.Format("03:04:05 PM"), entry)
}
