package padd

import "html/template"

// PageData holds data passed to templates for rendering
type PageData struct {
	Title            string                   // Page title - if an H1 (#) is present, it will be used, otherwise a metadata title will be used, finally the file name
	Description      string                   // Description from metadata
	Tags             []string                 // Tags from metadata (e.g. development, personal)
	Category         string                   // Category from metadata (e.g. work, personal)
	Status           string                   // Status from metadata (e.g. draft, in-progress, completed)
	StatusColor      string                   // Status color determined from MetadataConfig
	Priority         string                   // Priority from metadata (e.g. low, medium, high)
	PriorityColor    string                   // Priority color determined from MetadataConfig
	DueDate          string                   // Due date from metadata (if any)
	DueColor         string                   // Due date color determined from MetadataConfig
	TagColor         string                   // Tag color determined from MetadataConfig
	ContextColor     string                   // Context color determined from MetadataConfig
	CreatedAt        string                   // Created at from metadata (if any)
	UpdatedAt        string                   // Updated at from metadata (if any)
	Author           string                   // Author from metadata (if any)
	Contexts         []string                 // Contexts from metadata (e.g. @home, @work)
	SectionHeaders   []string                 // H2 headers in the current file for TOC
	CurrentFile      FileInfo                 // The current file info
	Content          template.HTML            // The rendered HTML content
	TasksTotal       int                      // Total number of tasks in the current file
	TasksCompleted   int                      // Total number of completed tasks in the current file
	TasksPending     int                      // Total number of pending tasks in the current file
	RawContent       string                   // The raw content of the current file
	IsEditing        bool                     // Whether the user is currently editing the file
	IsSearching      bool                     // Whether the user is currently searching the file
	IsResources      bool                     // Whether the current file is in the resources/ directory
	NavMenuFiles     []FileInfo               // List of file info objects for the navigation menu
	ArchiveType      string                   // "daily" or "journal" for archive pages
	SearchQuery      string                   // The current search query, if any
	SearchResults    map[string][]SearchMatch // Search results for the current query
	FlashMessage     string                   // Flash message to display
	FlashMessageType string                   // Flash message type
	ErrorMessage     string                   // Error message to display
	SearchMatch      int                      // To indicate which match in the line to highlight
	DirectoryTree    *DirectoryNode           // Directory tree for a page. For instance, resources or temporal archive pages.
	PADDVersion      string                   // The current version of PADD
	PADDDataDir      string                   // The current data directory for PADD
}

func (p PageData) HasTasks() bool {
	return p.TasksTotal > 0
}

func (p PageData) HasCompletedTasks() bool {
	return p.TasksCompleted > 0
}
