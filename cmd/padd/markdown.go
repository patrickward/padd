package main

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	pextension "github.com/patrickward/padd/extension"
)

type RenderedContent struct {
	Title             string         // The extracted title, if any.
	HTML              template.HTML  // The rendered HTML content.
	SectionHeaders    []string       // List of section headers (H2).
	TasksCount        int            // Number of tasks in the content.
	HasTasks          bool           // Indicates if the content contains task lists.
	HasCompletedTasks bool           // Indicates if the content contains completed tasks.
	Metadata          map[string]any // Additional metadata extracted from front matter.
}

func createMarkdownRenderer(dirManager *DirectoryManager) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(
			//extension.GFM,
			extension.Linkify,
			extension.Table,
			extension.Strikethrough,
			extension.Typographer,
			extension.DefinitionList,
			pextension.TaskList,
			pextension.NewIconExtension(pextension.NewDefaultIconChecker(dirManager)),
			meta.Meta,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			//parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML, but sanitize later
		),
	)
}

// determineTitle decides the title to use based on extracted title and metadata.
// Priority between preprocessing title given and metadata "title" should be:
// 1. If title from preprocessing is non-empty, use it.
// 2. Else if metadata contains a non-empty "title", use that.
// 3. Otherwise, return an empty string and the file name will be used as a fallback.
func renderedTitle(title string, metadata map[string]any) string {
	if strings.TrimSpace(title) != "" {
		return title
	}

	if metaTitle, ok := metadata["title"].(string); ok && metaTitle != "" {
		return metaTitle
	}
	return ""
}

// renderMarkdown converts markdown content to HTML with shortcode processing
func (s *Server) renderMarkdown(content string) RenderedContent {
	processResult := s.preProcessContent(content)

	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := s.md.Convert([]byte(processResult.Content), &buf, parser.WithContext(ctx)); err != nil {
		content = s.sanitizer.Sanitize(content)
		metadata := meta.Get(ctx)
		return RenderedContent{
			Title:             renderedTitle(processResult.Title, metadata),
			HTML:              template.HTML(fmt.Sprintf("<pre>%s</pre>", template.HTMLEscapeString(content))),
			SectionHeaders:    processResult.SectionHeaders,
			TasksCount:        pextension.TasksCount(ctx),
			HasTasks:          ctx.Get(pextension.HasTasksKey) == true,
			HasCompletedTasks: ctx.Get(pextension.HasCompletedTasksKey) == true,
			Metadata:          metadata,
		}
	}

	// Process inline svg images to ensure they are displayed correctly
	processedContent := s.postProcessContent(buf.String())
	processedContent = s.sanitizer.Sanitize(processedContent)
	metadata := meta.Get(ctx)

	return RenderedContent{
		Title:             renderedTitle(processResult.Title, metadata),
		HTML:              template.HTML(processedContent),
		SectionHeaders:    processResult.SectionHeaders,
		TasksCount:        pextension.TasksCount(ctx),
		HasTasks:          ctx.Get(pextension.HasTasksKey) == true,
		HasCompletedTasks: ctx.Get(pextension.HasCompletedTasksKey) == true,
		Metadata:          metadata,
	}
}

// renderMarkdownWithHighlight converts markdown content to HTML and highlights search query matches
func (s *Server) renderMarkdownWithHighlight(content, query string, targetIndex int) RenderedContent {
	if query == "" || targetIndex < 1 {
		return s.renderMarkdown(content)
	}

	processResult := s.preProcessContent(content)

	lines := strings.Split(processResult.Content, "\n")
	queryLower := strings.ToLower(query)
	matchIndex := 1

	// Find any frontmatter and remove it from content to avoid false matches,
	// but save it for adding back later
	var frontmatterLines []string
	if len(lines) > 0 && strings.HasPrefix(lines[0], "---") {
		frontmatterLines = append(frontmatterLines, lines[0])
		lines = lines[1:]
		for i, line := range lines {
			frontmatterLines = append(frontmatterLines, line)
			if strings.HasPrefix(line, "---") {
				lines = lines[i+1:]
				break
			}
		}
	}

	// Process each line and add match IDs
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			// Check if the line starts with list markers
			trimmed := strings.TrimLeft(line, " \t")
			var listMarker, content string

			if strings.HasPrefix(trimmed, "- ") {
				prefixLen := len(line) - len(trimmed) + 2 // account for "- "
				listMarker = line[:prefixLen]
				content = line[prefixLen:]
			} else if strings.HasPrefix(trimmed, "* ") {
				prefixLen := len(line) - len(trimmed) + 2 // account for "* "
				listMarker = line[:prefixLen]
				content = line[prefixLen:]
			} else {
				listMarker = ""
				content = line
			}

			// Apply highlighting to the content part only
			if matchIndex == targetIndex {
				// Add an ID to the line for scrolling
				lines[i] = listMarker + fmt.Sprintf(`<span id="search-match-%d" class="search-highlight search-target">%s</span>`, matchIndex, content)
			} else {
				lines[i] = listMarker + fmt.Sprintf(`<span id="search-match-%d" class="search-highlight">%s</span>`, matchIndex, content)
			}

			matchIndex++
		}
	}

	modifiedContent := strings.Join(lines, "\n")

	if len(frontmatterLines) > 0 {
		modifiedContent = strings.Join(frontmatterLines, "\n") + "\n" + modifiedContent
	}

	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := s.md.Convert([]byte(modifiedContent), &buf, parser.WithContext(ctx)); err != nil {
		content = s.sanitizer.Sanitize(content)
		metadata := meta.Get(ctx)

		return RenderedContent{
			Title:             renderedTitle(processResult.Title, metadata),
			HTML:              template.HTML(fmt.Sprintf("<pre>%s</pre>", template.HTMLEscapeString(content))),
			SectionHeaders:    processResult.SectionHeaders,
			TasksCount:        pextension.TasksCount(ctx),
			HasTasks:          ctx.Get(pextension.HasTasksKey) == true,
			HasCompletedTasks: ctx.Get(pextension.HasCompletedTasksKey) == true,
			Metadata:          metadata,
		}
	}

	// Process inline svg images to ensure they are displayed correctly
	processedContent := s.postProcessContent(buf.String())
	processedContent = s.sanitizer.Sanitize(processedContent)
	metadata := meta.Get(ctx)

	return RenderedContent{
		Title:             renderedTitle(processResult.Title, metadata),
		HTML:              template.HTML(processedContent),
		SectionHeaders:    processResult.SectionHeaders,
		TasksCount:        pextension.TasksCount(ctx),
		HasTasks:          ctx.Get(pextension.HasTasksKey) == true,
		HasCompletedTasks: ctx.Get(pextension.HasCompletedTasksKey) == true,
		Metadata:          metadata,
	}
}

// stripMarkdownHeaders removes leading header markers (#, ##, ###, ####) and whitespace from a line
func stripMarkdownHeaders(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	// Strip header markers (anything starting with #, ##, ###, or ####)
	trimmed = strings.TrimLeft(trimmed, "#")
	trimmed = strings.TrimLeft(trimmed, " \t")
	return trimmed
}

// stripMarkdownMarkers removes leading list markers (-, *) and whitespace from a line
func stripMarkdownMarkers(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") {
		return stripMarkdownHeaders(strings.TrimPrefix(trimmed, "- "))
	}
	if strings.HasPrefix(trimmed, "* ") {
		return stripMarkdownHeaders(strings.TrimPrefix(trimmed, "* "))
	}

	return stripMarkdownHeaders(trimmed)
}
