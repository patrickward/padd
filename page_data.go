package padd

import "html/template"

// PageData holds data passed to templates for rendering
type PageData struct {
	Title             string   // Page title - if an H1 (#) is present, it will be used, otherwise a metadata title will be used, finally the file name
	Description       string   // Description from metadata
	Tags              []string // Tags from metadata (e.g. development, personal)
	Category          string   // Category from metadata (e.g. work, personal)
	Status            string   // Status from metadata (e.g. draft, in-progress, completed)
	StatusColor       string   // Status color determined from MetadataConfig
	Priority          string   // Priority from metadata (e.g. low, medium, high)
	PriorityColor     string   // Priority color determined from MetadataConfig
	DueDate           string   // Due date from metadata (if any)
	DueColor          string   // Due date color determined from MetadataConfig
	TagColor          string   // Tag color determined from MetadataConfig
	ContextColor      string   // Context color determined from MetadataConfig
	CreatedAt         string   // Created at from metadata (if any)
	UpdatedAt         string   // Updated at from metadata (if any)
	Author            string   // Author from metadata (if any)
	Contexts          []string // Contexts from metadata (e.g. @home, @work)
	SectionHeaders    []string // H2 headers in the current file for TOC
	CurrentFile       FileInfo
	Content           template.HTML
	TasksCount        int  // Total number of tasks in the current file
	HasCompletedTasks bool // Whether the current file has completed tasks
	RawContent        string
	IsEditing         bool
	IsSearching       bool
	IsResources       bool
	NavMenuFiles      []FileInfo
	ResourceTree      *DirectoryNode
	TemporalYears     []string
	TemporalFiles     map[string][]FileInfo
	ArchiveType       string // "daily" or "journal" for archive pages
	SearchQuery       string
	SearchResults     map[string][]SearchMatch
	FlashMessage      string
	FlashMessageType  string
	ErrorMessage      string
	SearchMatch       int // To indicate which match in the line to highlight
}

func (p PageData) HasTasks() bool {
	return p.TasksCount > 0
}
