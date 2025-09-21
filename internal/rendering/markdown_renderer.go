package rendering

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/patrickward/padd"
	pextension "github.com/patrickward/padd/extension"
	"github.com/patrickward/padd/internal/contentutil"
	"github.com/patrickward/padd/internal/files"
)

type MarkdownRenderer struct {
	md            goldmark.Markdown
	sanitizer     *bluemonday.Policy
	fileRepo      *files.FileRepository
	rootManager   *files.RootManager
	preprocessor  *MarkdownPreprocessor
	postprocessor *MarkdownPostprocessor
}

type RenderedContent struct {
	Title          string         // The extracted title, if any.
	HTML           template.HTML  // The rendered HTML content.
	SectionHeaders []string       // List of section headers (H2).
	TasksTotal     int            // Number of tasks in the content.
	TasksCompleted int            // Number of completed tasks in the content.
	TasksPending   int            // Number of pending tasks in the content.
	Metadata       map[string]any // Additional metadata extracted from front matter.
}

type RenderOptions struct {
	SearchQuery  string
	TargetIndex  int
	EnableSearch bool
}

// NewMarkdownRenderer creates a new MarkdownRenderer instance.
func NewMarkdownRenderer(rootManager *files.RootManager, fileRepo *files.FileRepository) *MarkdownRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(
			//extension.GFM,
			extension.Linkify,
			extension.Table,
			extension.Strikethrough,
			extension.Typographer,
			extension.DefinitionList,
			pextension.TaskList,
			pextension.NewIconExtension(pextension.NewDefaultIconChecker(rootManager, padd.StaticFS)),
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

	sanitizer := createSanitizerPolicy()

	return &MarkdownRenderer{
		md:            md,
		sanitizer:     sanitizer,
		fileRepo:      fileRepo,
		rootManager:   rootManager,
		preprocessor:  NewMarkdownPreprocessor(fileRepo),
		postprocessor: NewMarkdownPostprocessor(rootManager),
	}
}

// Render renders the given Markdown content.
func (mr *MarkdownRenderer) Render(content string) RenderedContent {
	return mr.renderWithOptions(content, RenderOptions{})
}

// RenderWithHighlight renders the given Markdown content with search highlighting.
func (mr *MarkdownRenderer) RenderWithHighlight(content string, query string, targetIndex int) RenderedContent {
	opts := RenderOptions{
		SearchQuery:  query,
		TargetIndex:  targetIndex,
		EnableSearch: true,
	}

	return mr.renderWithOptions(content, opts)
}

// renderWithOptions renders the given Markdown content with the given options.
func (mr *MarkdownRenderer) renderWithOptions(content string, opts RenderOptions) RenderedContent {
	processResult := mr.preprocessor.Process(content)

	// Apply search highlighting if enabled
	if opts.EnableSearch && opts.SearchQuery != "" {
		processResult.Content = mr.applySearchHighlighting(processResult.Content, opts)
	}

	// Convert to HTML
	var buf bytes.Buffer
	ctx := parser.NewContext()
	if err := mr.md.Convert([]byte(processResult.Content), &buf, parser.WithContext(ctx)); err != nil {
		return mr.renderError(ctx, content, processResult, err)
	}

	// Post-process HTML
	processedHTML := mr.postprocessor.Process(buf.String())
	processedHTML = mr.sanitizer.Sanitize(processedHTML)
	metadata := meta.Get(ctx)

	taskStats := pextension.TaskStats(ctx)

	return RenderedContent{
		Title:          renderedTitle(processResult.Title, metadata),
		HTML:           template.HTML(processedHTML),
		SectionHeaders: processResult.SectionHeaders,
		TasksTotal:     taskStats.Total,
		TasksCompleted: taskStats.Completed,
		TasksPending:   taskStats.Pending,
		Metadata:       metadata,
	}
}

// applySearchHighlighting applies search highlighting to the given content.
func (mr *MarkdownRenderer) applySearchHighlighting(content string, opts RenderOptions) string {
	lines := contentutil.SplitLines(content)
	queryLower := strings.ToLower(opts.SearchQuery)
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
			if matchIndex == opts.TargetIndex {
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

	return modifiedContent
}

// renderError renders an error message for the given content.
func (mr *MarkdownRenderer) renderError(ctx parser.Context, content string, processResult PreprocessingResult, err error) RenderedContent {
	content = mr.sanitizer.Sanitize(content)
	metadata := meta.Get(ctx)

	// Prepend the error message to the content
	content = fmt.Sprintf("<div class=\"callout danger\">%s</div><pre>%s</pre>", err.Error(), template.HTMLEscapeString(content))

	taskStats := pextension.TaskStats(ctx)

	return RenderedContent{
		Title:          renderedTitle(processResult.Title, metadata),
		HTML:           template.HTML(content),
		SectionHeaders: processResult.SectionHeaders,
		TasksTotal:     taskStats.Total,
		TasksCompleted: taskStats.Completed,
		TasksPending:   taskStats.Pending,
		Metadata:       metadata,
	}
}

// createSanitizerPolicy creates a new sanitizer policy for HTML rendering.
func createSanitizerPolicy() *bluemonday.Policy {
	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowAttrs("class", "id").OnElements("span", "div", "i", "code", "pre", "p", "h1", "h2", "h3", "h4", "h5", "h6")

	// Allow form elements, so we can use them in markdown for checklists, etc.
	sanitizer.AllowElements("form", "input", "textarea", "button", "select", "option", "label")
	sanitizer.AllowAttrs("type", "checked", "disabled", "name", "value", "placeholder").OnElements("input", "textarea", "button", "select", "option", "label")

	// Allow all of the "hx-*" attributes for htmx (https://htmx.org/)
	sanitizer.AllowAttrs("hx-get", "hx-post", "hx-put", "hx-delete", "hx-patch", "hx-target", "hx-swap", "hx-trigger", "hx-vals", "hx-include", "hx-headers", "hx-push-url", "hx-confirm", "hx-indicator", "hx-params").
		OnElements("a", "form", "button", "input", "select", "textarea", "div", "span", "p")

	// Allow media elements
	// "audio" "svg" "video" are all permitted
	sanitizer.AllowElements("audio", "svg", "video")
	sanitizer.AllowAttrs("autoplay", "controls", "loop", "muted", "preload", "src", "type", "width", "height").OnElements("audio", "video")
	sanitizer.AllowAttrs("xmlns", "viewbox", "width", "height", "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin").OnElements("svg", "path", "circle", "rect", "line", "polyline", "polygon")
	sanitizer.AllowAttrs("d", "cx", "cy", "r", "x", "y", "x1", "y1", "x2", "y2", "points").OnElements("path", "circle", "rect", "line", "polyline", "polygon")
	return sanitizer
}

// renderedTitle decides the title to use based on extracted title and metadata.
// Priority between preprocessing title given and metadata "title" should be:
// 1. If the title from preprocessing is non-empty, use it.
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

// StripMarkdownHeaders removes leading header markers (#, ##, ###, ####) and whitespace from a line
func StripMarkdownHeaders(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	// Strip header markers (anything starting with #, ##, ###, or ####)
	trimmed = strings.TrimLeft(trimmed, "#")
	trimmed = strings.TrimLeft(trimmed, " \t")
	return trimmed
}

// StripMarkdownMarkers removes leading list markers (-, *) and whitespace from a line
func StripMarkdownMarkers(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") {
		return StripMarkdownHeaders(strings.TrimPrefix(trimmed, "- "))
	}
	if strings.HasPrefix(trimmed, "* ") {
		return StripMarkdownHeaders(strings.TrimPrefix(trimmed, "* "))
	}

	return StripMarkdownHeaders(trimmed)
}
