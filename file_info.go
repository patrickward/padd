package padd

import (
	"html/template"
	"strings"
)

// FileInfo represents metadata about a markdown file
type FileInfo struct {
	ID          string
	Path        string
	Display     string
	DisplayBase string // Base name without directory
	IsCurrent   bool
	IsTemporal  bool   // True if the file is a temporal file (daily/journal)
	IsNavActive bool   // True if the file should indicate active in navigation
	Directory   string // Directory path relative to the resources/ (empty for core and files at the root of resources/)
	Depth       int    // Depth in the resources/ directory structure (0 for core and files at the root of resources/)
	IsResource  bool   // True if the file is in the resources/ directory
	Year        string // Year extracted from the path if applicable
	Month       string // Month extracted from the path if applicable
	MonthName   string // Full month name extracted from the path if applicable
}

// RelativePath returns the file path relative to the resources/ directory if applicable
func (f FileInfo) RelativePath() string {
	if f.IsResource {
		return f.Path[len("resources/"):]
	}

	return f.Path
}

// CSSClass generates a safe CSS class name based on the file ID
func (f FileInfo) CSSClass() string {
	id := f.ID
	// Replace specific characters to ensure it's a valid CSS ID (e.g. no slashes, dots, etc.)
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, "\\", "-")
	id = strings.ReplaceAll(id, ".", "-")
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	// IDs should not contain consecutive dashes
	id = strings.ReplaceAll(id, "--", "-")
	// Trim leading or trailing dashes
	id = strings.Trim(id, "-")
	// Escape to ensure it's safe for HTML
	id = template.HTMLEscapeString(id)
	id = template.JSEscapeString(id)
	id = template.URLQueryEscaper(id)
	return id
}
