package main

import (
	"net/http"
	"strings"

	"github.com/patrickward/padd"
)

type searchResults map[string][]padd.SearchMatch

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		s.redirectTo(w, r, "/")
		return
	}

	results := make(searchResults)

	// Search core files
	for _, file := range s.fileRepo.CoreFiles() {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	// Search resource files
	resourceDir := s.fileRepo.DirectoryTreeFor(s.fileRepo.Config().ResourcesDirectory)
	s.searchDirectory(query, resourceDir, results)

	// Search temporal files
	temporalDirectories := s.fileRepo.Config().TemporalDirectories()
	for _, dir := range temporalDirectories {
		node := s.fileRepo.DirectoryTreeFor(dir)
		s.searchDirectory(query, node, results)
	}

	data := padd.PageData{
		Title:         "Search Results",
		IsSearching:   true,
		SearchQuery:   query,
		SearchResults: results,
		NavMenuFiles:  s.navigationMenu(""),
	}

	if err := s.executePage(w, "search.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

// searchDirectory recursively searches a directory for matches to a query and adds to the results map
func (s *Server) searchDirectory(query string, directory *padd.DirectoryNode, results searchResults) {
	for _, file := range directory.Files {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	for _, child := range directory.Directories {
		s.searchDirectory(query, child, results)
	}
}

func (s *Server) searchFile(file padd.FileInfo, query string) []padd.SearchMatch {
	var matches []padd.SearchMatch
	content, err := s.rootManager.ReadFile(file.Path)
	if err != nil {
		return matches
	}

	lines := padd.SplitLines(string(content))
	matchIndex := 1 // To track the occurrence of matches in a line
	queryLower := strings.ToLower(query)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {

			cleanedLine := padd.StripMarkdownMarkers(line)
			renderedContent := s.renderer.Render(cleanedLine)
			matches = append(matches, padd.SearchMatch{
				LineNum:    i + 1,
				Line:       line,
				Rendered:   renderedContent.HTML,
				MatchIndex: matchIndex,
			})
			matchIndex++
		}
	}

	return matches
}
