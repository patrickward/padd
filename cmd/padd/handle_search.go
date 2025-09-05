package main

import (
	"net/http"
	"strings"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	results := make(map[string][]SearchMatch)

	// Search core files
	for _, file := range coreFilesMap {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	// Search resource files
	resourceFiles := s.getResourceFiles("")
	for _, file := range resourceFiles {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	// Search temporal files
	years, temporalFiles, err := s.getTemporalFiles("daily")
	if err == nil {
		for _, year := range years {
			for _, file := range temporalFiles[year] {
				if matches := s.searchFile(file, query); len(matches) > 0 {
					results[file.ID] = matches
				}
			}
		}
	}

	data := PageData{
		Title:         "Search Results",
		IsSearching:   true,
		SearchQuery:   query,
		SearchResults: results,
		CoreFiles:     s.getCoreFiles(""),
		ResourceFiles: s.getResourceFiles(""),
	}

	if err := s.executePage(w, "search.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}

func (s *Server) searchFile(file FileInfo, query string) []SearchMatch {
	var matches []SearchMatch
	content, err := s.rootManager.ReadFile(file.Path)
	if err != nil {
		return matches
	}

	lines := strings.Split(string(content), "\n")
	matchIndex := 1 // To track the occurrence of matches in a line
	queryLower := strings.ToLower(query)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {

			cleanedLine := stripMarkdownMarkers(line)
			renderedContent := s.renderMarkdown(cleanedLine)
			matches = append(matches, SearchMatch{
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
