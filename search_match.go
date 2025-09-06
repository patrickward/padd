package padd

import "html/template"

// SearchMatch represents a single line match in a file
type SearchMatch struct {
	LineNum    int           // The line number in the file (1-based)
	Line       string        // The raw line text
	Rendered   template.HTML // The rendered HTML of the line (for display)
	MatchIndex int           // The index of the match in the line, for potential highlighting
}
