package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// ProcessingResult holds the result of processing content.
type ProcessingResult struct {
	Title          string   // The extracted title, if any.
	Content        string   // The processed content.
	SectionHeaders []string // List of section headers (H2).
}

// preProcessContent processes the content to extract the title, section headers,
// and wiki links, prior to Markdown rendering.
func (s *Server) preProcessContent(content string) ProcessingResult {
	lines := strings.Split(content, "\n")

	// Compile regexes once for efficiency
	titleRe := regexp.MustCompile(`^#\s+(.+)$`)
	sectionsRe := regexp.MustCompile(`^##\s+(.+)$`)
	wikiLinksRe := regexp.MustCompile(`\[\[([^]\n]+)]]`)

	var title string
	var headers []string

	for i, line := range lines {
		// Process title
		if title == "" {
			if matches := titleRe.FindStringSubmatch(line); matches != nil {
				title = strings.TrimSpace(matches[1])
				lines[i] = "" // Remove the title line, as we'll it use it outside the content
				continue
			}
		}

		// Process Section headers
		if matches := sectionsRe.FindStringSubmatch(line); matches != nil {
			headers = append(headers, strings.TrimSpace(matches[1]))
			continue // Skip adding the header line to headers
		}

		// Process wiki links
		line = s.processWikiLinkShortcodes(line, wikiLinksRe)
		lines[i] = line
	}

	return ProcessingResult{
		Title:          title,
		Content:        strings.Join(lines, "\n"),
		SectionHeaders: headers,
	}
}

// processWikiLinkShortcodes processes wiki link shortcodes in the format [[Page Name]]
// and replaces them with appropriate links or not-found messages.
// TODO: Move to a proper goldmark extension?
func (s *Server) processWikiLinkShortcodes(line string, wikiRe *regexp.Regexp) string {
	// Process wiki links first
	line = wikiRe.ReplaceAllStringFunc(line, func(match string) string {
		pageName := strings.Trim(match, "[]")

		// Trim whitespace from the page name
		pageName = strings.TrimSpace(pageName)

		// Skip empty page names
		if pageName == "" {
			return match // Return original if empty
		}

		fileID := s.createID(pageName)

		// Check if the file exists
		if file, err := s.getFileInfo(fileID); err == nil {
			// File exists, return a link
			return fmt.Sprintf(`[%s](/%s)`, file.Display, file.ID)
		} else if file, err := s.getFileInfo(filepath.Join(resourcesDir, pageName)); err == nil {
			// File exists in resources, return a link
			return fmt.Sprintf(`[%s](/%s)`, file.Display, file.ID)
		}

		return fmt.Sprintf(`<span class="text-color danger">!! [[%s]] not found !!</span>`, pageName)
	})
	return line
}
