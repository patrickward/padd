package padd

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func TitleCase(s string) string {
	return cases.Title(language.English).String(s)
}

// normalizeLineEndings normalizes line endings in a string.
func normalizeLineEndings(content string) string {
	// Replace Windows CRLF
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Replace legacy Mac CR
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

// SplitLines splits a string into lines, normalizing line endings.
func SplitLines(content string) []string {
	return strings.Split(normalizeLineEndings(content), "\n")
}
