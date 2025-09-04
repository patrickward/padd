package main

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

// DirectoryNode represents a node in the directory tree
type DirectoryNode struct {
	Name        string
	Files       []FileInfo
	Directories map[string]*DirectoryNode
}

// PageData holds data passed to templates for rendering
type PageData struct {
	Title            string   // Page title - if an H1 (#) is present, it will be used, otherwise a metadata title will be used, finally the file name
	Description      string   // Description from metadata
	Tags             []string // Tags from metadata (e.g. development, personal)
	Category         string   // Category from metadata (e.g. work, personal)
	Status           string   // Status from metadata (e.g. draft, in-progress, completed)
	StatusColor      string   // Status color determined from MetadataConfig
	Priority         string   // Priority from metadata (e.g. low, medium, high)
	PriorityColor    string   // Priority color determined from MetadataConfig
	DueDate          string   // Due date from metadata (if any)
	DueColor         string   // Due date color determined from MetadataConfig
	TagColor         string   // Tag color determined from MetadataConfig
	ContextColor     string   // Context color determined from MetadataConfig
	CreatedAt        string   // Created at from metadata (if any)
	UpdatedAt        string   // Updated at from metadata (if any)
	Author           string   // Author from metadata (if any)
	Contexts         []string // Contexts from metadata (e.g. @home, @work)
	SectionHeaders   []string // H2 headers in the current file for TOC
	CurrentFile      FileInfo
	Content          template.HTML
	HasTasks         bool // Whether the current file has task lists
	RawContent       string
	IsEditing        bool
	IsSearching      bool
	IsResources      bool
	CoreFiles        []FileInfo
	ResourceFiles    []FileInfo
	ResourceTree     *DirectoryNode
	TemporalYears    []string
	TemporalFiles    map[string][]FileInfo
	ArchiveType      string // "daily" or "journal" for archive pages
	SearchQuery      string
	SearchResults    map[string][]SearchMatch
	FlashMessage     string
	FlashMessageType string
	ErrorMessage     string
	SearchMatch      int // To indicate which match in the line to highlight
}

// SearchMatch represents a single line match in a file
type SearchMatch struct {
	LineNum    int           // The line number in the file (1-based)
	Line       string        // The raw line text
	Rendered   template.HTML // The rendered HTML of the line (for display)
	MatchIndex int           // The index of the match in the line, for potential highlighting
}
