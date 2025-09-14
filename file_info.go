package padd

import (
	"strings"
)

// FileInfo represents metadata about a markdown file
type FileInfo struct {
	ID            string         // Unique ID for the file
	Path          string         // The full file path of the file as a string
	Title         string         // The name of the file as a string (may include the directory path)
	TitleBase     string         // The title case of th file name without the directory path
	DirectoryPath string         // The parent directory path of the file as a string
	DirectoryNode *DirectoryNode // If the file is a directory, this is the directory node in the directory tree
	Depth         int            // Depth in the resources/ directory structure (0 for core and files at the root of resources/)
	IsTemporal    bool           // True if the file is a temporal file (daily/journal)
	IsNavActive   bool           // True if the file should indicate active in navigation
	IsResource    bool           // True if the file is in the resources/ directory
	IsDirectory   bool           // True if the file is a directory
}

// RelativePath returns the file path relative to the resources/ directory if applicable
func (f FileInfo) RelativePath() string {
	if f.IsResource {
		return f.Path[len("resources/"):]
	} else if f.IsTemporal {
		// Remove daily/ or journal/ from the path
		if strings.HasPrefix(f.Path, "daily/") {
			return f.Path[len("daily/"):]
		} else if strings.HasPrefix(f.Path, "journal/") {
			return f.Path[len("journal/"):]
		}
	}

	return f.Path
}

// PathParts returns the file path parts
func (f FileInfo) PathParts() []string {
	return strings.Split(f.Path, "/")
}

// RelativePathParts returns the file path parts relative to the resources/ directory if applicable
func (f FileInfo) RelativePathParts() []string {
	return strings.Split(f.RelativePath(), "/")
}

type Breadcrumb struct {
	Path    string
	Name    string
	IsFirst bool
	IsLast  bool
}

// BreadcrumbParts returns the breadcrumb parts for a file's path.
func (f FileInfo) BreadcrumbParts() []Breadcrumb {
	parts := f.PathParts()
	var breadcrumbs []Breadcrumb
	for i, part := range parts {
		isLast := i == len(parts)-1
		breadcrumbs = append(breadcrumbs, Breadcrumb{
			Path:    "/" + strings.Join(parts[:i+1], "/"),
			Name:    TitleCase(part),
			IsFirst: i == 0,
			IsLast:  isLast,
		})
	}

	return breadcrumbs
}

func (f FileInfo) IsEmpty() bool {
	return f.ID == ""
}

func (f FileInfo) Year() string {
	if !f.IsTemporal {
		return ""
	}

	parts := strings.Split(f.Path, "/")
	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}

func (f FileInfo) Month() string {
	if !f.IsTemporal {
		return ""
	}

	parts := strings.Split(f.Path, "/")
	if len(parts) >= 3 {
		monthFile := strings.TrimSuffix(parts[2], ".md")
		monthParts := strings.SplitN(monthFile, "-", 2)
		if len(monthParts) >= 1 {
			return monthParts[0] // 09-september -> 09
		}
	}

	return ""
}

func (f FileInfo) MonthName() string {
	if !f.IsTemporal {
		return ""
	}

	parts := strings.Split(f.Path, "/")
	if len(parts) >= 3 {
		monthFile := strings.TrimSuffix(parts[2], ".md")
		monthParts := strings.SplitN(monthFile, "-", 2)
		if len(monthParts) >= 2 {
			return TitleCase(monthParts[1]) // 09-september -> September
		}
	}

	return ""
}
