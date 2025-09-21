package rendering

import (
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/patrickward/padd"
	"github.com/patrickward/padd/internal/files"
)

// MarkdownPostprocessor represents a postprocessor for markdown files
// NOTE: some of this could be in an extension, but it's good enough for now
type MarkdownPostprocessor struct {
	rootManager *files.RootManager
}

// NewMarkdownPostprocessor creates a new MarkdownPostprocessor for the given RootManager
func NewMarkdownPostprocessor(rootManager *files.RootManager) *MarkdownPostprocessor {
	return &MarkdownPostprocessor{rootManager: rootManager}
}

// Process performs the postprocessing of the given Markdown content
func (mp *MarkdownPostprocessor) Process(content string) string {
	return mp.processInlineSVG(content)
}

func (mp *MarkdownPostprocessor) processInlineSVG(htmlContent string) string {
	// Replace <img> tags with inline SVG content
	re := regexp.MustCompile(`<img[^>]+src="([^">]+\.svg)"[^>]*>`)

	return re.ReplaceAllStringFunc(htmlContent, func(imgTag string) string {
		// Extract the icon path
		srcMatch := regexp.MustCompile(`src="([^">]+\.svg)"`).FindStringSubmatch(imgTag)
		if len(srcMatch) < 2 {
			return imgTag // No src found, return original tag
		}

		iconPath := strings.TrimPrefix(srcMatch[1], "/images/")
		svgContent := mp.getInlineSVG(iconPath)
		if svgContent != "" {
			return svgContent
		}

		return imgTag // Return original tag if SVG not found
	})
}

func (mp *MarkdownPostprocessor) getInlineSVG(iconPath string) string {
	// Try user's path first
	userSVGPath := filepath.Join("images", iconPath)
	if mp.rootManager.FileExists(userSVGPath) {
		content, err := mp.rootManager.ReadFile(userSVGPath)
		if err == nil {
			return string(content)
		}
	}

	// Fallback to static embedded files
	staticPath := "static/images/" + iconPath
	if file, err := padd.StaticFS.Open(staticPath); err == nil {
		defer func(file fs.File) {
			_ = file.Close()
		}(file)

		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			content, err := io.ReadAll(file)
			if err == nil {
				return string(content)
			}
		}
	}

	return ""
}
