package padd

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type PreprocessingResult struct {
	Title          string
	Content        string
	SectionHeaders []string
}

// MarkdownPreprocessor represents a Markdown preprocessor for markdown files
// NOTE: some of this could be in an extension, but it's good enough for now
type MarkdownPreprocessor struct {
	fileRepo *FileRepository
}

// NewMarkdownPreprocessor creates a new MarkdownPreprocessor for the given RootManager
func NewMarkdownPreprocessor(fileRepo *FileRepository) *MarkdownPreprocessor {
	return &MarkdownPreprocessor{fileRepo: fileRepo}
}

func (mp *MarkdownPreprocessor) Process(content string) PreprocessingResult {
	lines := SplitLines(content)

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
		line = mp.processWikiLinkShortcodes(line, wikiLinksRe)
		lines[i] = line
	}

	return PreprocessingResult{
		Title:          title,
		Content:        strings.Join(lines, "\n"),
		SectionHeaders: headers,
	}
}

// processWikiLinkShortcodes processes wiki link shortcodes in the format [[Page Name]]
// and replaces them with appropriate links or not-found messages.
// TODO: Move to a proper goldmark extension?
func (mp *MarkdownPreprocessor) processWikiLinkShortcodes(line string, wikiRe *regexp.Regexp) string {
	// Process wiki links first
	line = wikiRe.ReplaceAllStringFunc(line, func(match string) string {
		pageName := strings.Trim(match, "[]")

		// Trim whitespace from the page name
		pageName = strings.TrimSpace(pageName)

		// Skip empty page names
		if pageName == "" {
			return match // Return original if empty
		}

		fileID := mp.fileRepo.CreateID(pageName)

		// Check if the file exists
		if file, err := mp.fileRepo.FileInfo(fileID); err == nil {
			// File exists, return a link
			return fmt.Sprintf(`[%s](/%s)`, file.Title, file.ID)
		} else if file, err := mp.fileRepo.FileInfo(filepath.Join(mp.fileRepo.Config().ResourcesDirectory, pageName)); err == nil {
			// File exists in resources, return a link
			return fmt.Sprintf(`[%s](/%s)`, file.Title, file.ID)
		}

		return fmt.Sprintf(`<span class="text-color danger">!! [[%s]] not found !!</span>`, pageName)
	})

	return line
}
