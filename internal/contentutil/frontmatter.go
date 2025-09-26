package contentutil

import (
	"strings"
)

type FrontmatterBounds struct {
	Start int
	End   int
	Found bool
}

// FindFrontmatter finds the bounds of the frontmatter section in the given lines
func FindFrontmatter(lines []string) FrontmatterBounds {
	if len(lines) == 0 {
		return FrontmatterBounds{}
	}

	// Skip any blank lines at the start
	startIdx := 0
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}

	// Return empty bounds if no frontmatter found, i.e., the first non-blank does not start with "---"
	if startIdx >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[startIdx]), "---") {
		return FrontmatterBounds{}
	}

	// Look for the closing frontmatter delimiter
	for i := startIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Closing delimiter found
		if strings.HasPrefix(line, "---") {
			return FrontmatterBounds{
				Start: startIdx,
				End:   i + 1,
				Found: true,
			}
		}
	}

	// Return empty bounds if no closing delimiter found
	return FrontmatterBounds{}
}
