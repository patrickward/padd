package padd

import "strings"

type FrontmatterBounds struct {
	Start int
	End   int
	Found bool
}

// findFrontmatter finds the bounds of the frontmatter section in the given lines
func findFrontmatter(lines []string) FrontmatterBounds {
	if len(lines) == 0 {
		return FrontmatterBounds{}
	}

	// Skip any blank lines at the start
	startIdx := 0
	for startIdx < len(lines) && strings.TrimSpace(lines[startIdx]) == "" {
		startIdx++
	}

	// Return empty bounds if no frontmatter found, i.e., the first non-blank line is not "---"
	if startIdx >= len(lines) || strings.TrimSpace(lines[startIdx]) != "---" {
		return FrontmatterBounds{}
	}

	// Look for the closing frontmatter delimiter
	for i := startIdx + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Closing delimiter found
		if line == "---" {
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
