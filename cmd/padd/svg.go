package main

import (
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/patrickward/padd"
)

func (s *Server) processInlineSVG(htmlContent string) string {
	// Replace <img> tags with inline SVG content
	re := regexp.MustCompile(`<img[^>]+src="([^">]+\.svg)"[^>]*>`)

	return re.ReplaceAllStringFunc(htmlContent, func(imgTag string) string {
		// Extract the icon path
		srcMatch := regexp.MustCompile(`src="([^">]+\.svg)"`).FindStringSubmatch(imgTag)
		if len(srcMatch) < 2 {
			return imgTag // No src found, return original tag
		}

		iconPath := strings.TrimPrefix(srcMatch[1], "/images/")
		svgContent := s.getInlineSVG(iconPath)
		if svgContent != "" {
			return svgContent
		}

		return imgTag // Return original tag if SVG not found
	})
}

func (s *Server) getInlineSVG(iconPath string) string {
	// Try user's path first
	userSVGPath := filepath.Join("images", iconPath)
	if s.rootManager.FileExists(userSVGPath) {
		content, err := s.rootManager.ReadFile(userSVGPath)
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
