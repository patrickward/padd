package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"
)

func (s *Server) processShortcodes(content string) string {
	// Process line by line to avoid cross-line matching
	lines := strings.Split(content, "\n")

	// Compile regexes once for efficiency
	wikiRe := regexp.MustCompile(`\[\[([^]\n]+)]]`)
	iconRe := regexp.MustCompile(`::([a-zA-Z0-9\-_]+)::`)

	for i, line := range lines {
		line = s.processWikiLinkShortcodes(line, wikiRe)
		line = s.processIconShortcodes(line, iconRe)
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}

func (s *Server) processIconShortcodes(line string, iconRe *regexp.Regexp) string {
	// Then process icon shortcodes
	line = iconRe.ReplaceAllStringFunc(line, func(match string) string {
		iconName := strings.Trim(match, ":")

		// Trim whitespace and validate icon name
		iconName = strings.TrimSpace(iconName)

		// Skip empty icon names
		if iconName == "" {
			return match // Return original if empty
		}

		if s.iconExists(iconName) {
			return fmt.Sprintf(`<span class="icon">![%s](/images/icons/%s.svg)</span>`, strings.TrimSuffix(iconName, ".svg"), iconName)
		}

		// If the icon doesn't exist, return the original text
		return match
	})
	return line
}

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

func (s *Server) iconExists(iconName string) bool {
	// Add .svg if not present
	if !strings.HasSuffix(iconName, ".svg") {
		iconName = iconName + ".svg"
	}

	// Check user's path first
	userSVGPath := filepath.Join("images", "icons", iconName)
	if s.dirManager.Exists(userSVGPath) {
		return true
	}

	// Fallback to static embedded files
	staticPath := "static/images/icons/" + iconName
	if file, err := staticFS.Open(staticPath); err == nil {
		defer func(file fs.File) {
			_ = file.Close()
		}(file)

		if stat, err := file.Stat(); err == nil && !stat.IsDir() {
			return true
		}
	}

	return false
}
