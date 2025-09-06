package padd

// DirectoryNode represents a node in the directory tree
type DirectoryNode struct {
	Name        string
	Files       []FileInfo
	Directories map[string]*DirectoryNode
}
