package padd

import "strings"

// DirectoryNode represents a node in the directory tree
type DirectoryNode struct {
	Name        string
	Files       []FileInfo
	Directories map[string]*DirectoryNode
}

// IsEmpty returns true if the directory node has no files or subdirectories
func (dn *DirectoryNode) IsEmpty() bool {
	return len(dn.Files) == 0 && len(dn.Directories) == 0
}

func (dn *DirectoryNode) FindFile(id string) FileInfo {
	// Search files by ID
	for i := range dn.Files {
		if dn.Files[i].ID == id {
			return dn.Files[i]
		}
	}

	// Search within subdirectories
	for _, child := range dn.Directories {
		if file := child.FindFile(id); file.ID != "" {
			return file
		}
	}

	return FileInfo{}
}

func (dn *DirectoryNode) FindDirectory(path string) *DirectoryNode {
	if path == "" {
		return dn
	}

	parts := strings.Split(path, "/")
	currentNode := dn

	for _, part := range parts {
		if child, exists := currentNode.Directories[part]; exists {
			currentNode = child
		} else {
			return nil
		}
	}

	return currentNode
}
