package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// DirectoryManager provides safe filesystem operations within a specific directory using os.Root
type DirectoryManager struct {
	path string
}

// NewDirectoryManager creates a new DirectoryManager for the given directory path
func NewDirectoryManager(path string) (*DirectoryManager, error) {
	// Ensure directory exists
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	// Test that we can open the directory as a root
	testRoot, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open directory as root %s: %w", path, err)
	}
	_ = testRoot.Close()

	return &DirectoryManager{path: path}, nil
}

// withRoot executes a function with a safely opened os.Root
func (dm *DirectoryManager) withRoot(fn func(*os.Root) error) error {
	root, err := os.OpenRoot(dm.path)
	if err != nil {
		return fmt.Errorf("failed to open root: %w", err)
	}
	defer func(root *os.Root) {
		_ = root.Close()
	}(root)

	return fn(root)
}

// ReadFile reads the contents of a file using Root.ReadFile
func (dm *DirectoryManager) ReadFile(filename string) ([]byte, error) {
	var content []byte
	err := dm.withRoot(func(root *os.Root) error {
		var err error
		content, err = root.ReadFile(filename)
		return err
	})
	return content, err
}

// WriteFile writes content to a file using Root.WriteFile
func (dm *DirectoryManager) WriteFile(filename string, content []byte, perm os.FileMode) error {
	return dm.withRoot(func(root *os.Root) error {
		return root.WriteFile(filename, content, perm)
	})
}

// WriteString writes a string to a file
func (dm *DirectoryManager) WriteString(filename string, content string) error {
	return dm.WriteFile(filename, []byte(content), 0644)
}

// FileExists checks if a file exists using Root.Stat
func (dm *DirectoryManager) FileExists(filename string) bool {
	exists := false
	_ = dm.withRoot(func(root *os.Root) error {
		_, err := root.Stat(filename)
		exists = err == nil
		return nil
	})
	return exists
}

// Stat returns file info using Root.Stat
func (dm *DirectoryManager) Stat(filename string) (os.FileInfo, error) {
	var info os.FileInfo
	err := dm.withRoot(func(root *os.Root) error {
		var err error
		info, err = root.Stat(filename)
		return err
	})
	return info, err
}

// MkdirAll creates a directory and any necessary parent directories using Root.MkdirAll
func (dm *DirectoryManager) MkdirAll(dir string, perm os.FileMode) error {
	return dm.withRoot(func(root *os.Root) error {
		return root.MkdirAll(dir, perm)
	})
}

// Remove removes a file using Root.Remove
func (dm *DirectoryManager) Remove(filename string) error {
	return dm.withRoot(func(root *os.Root) error {
		return root.Remove(filename)
	})
}

// RemoveAll removes a directory and all its contents using Root.RemoveAll
func (dm *DirectoryManager) RemoveAll(path string) error {
	return dm.withRoot(func(root *os.Root) error {
		return root.RemoveAll(path)
	})
}

// WalkDir walks the directory tree using Root.FS()
func (dm *DirectoryManager) WalkDir(root string, fn fs.WalkDirFunc) error {
	return dm.withRoot(func(osRoot *os.Root) error {
		fsys := osRoot.FS()
		return fs.WalkDir(fsys, root, fn)
	})
}

// CreateFileIfNotExists creates a file with default content if it doesn't exist
func (dm *DirectoryManager) CreateFileIfNotExists(filename string, defaultContent string) error {
	if dm.FileExists(filename) {
		return nil
	}
	return dm.WriteString(filename, defaultContent)
}

// ScanResult holds information about a scanned file or directory
type ScanResult struct {
	Path         string
	Name         string
	IsDir        bool
	RelativePath string
}

// Scan scans the directory tree starting from rootDir, applying an optional filter function
func (dm *DirectoryManager) Scan(rootDir string, filter func(string, fs.DirEntry) bool) ([]ScanResult, error) {
	var results []ScanResult

	err := dm.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking despite errors
		}

		if filter != nil && !filter(path, d) {
			return nil
		}

		relativePath := path
		if rootDir != "." && rootDir != "" {
			relativePath = strings.TrimPrefix(path, rootDir+string(filepath.Separator))
		}

		results = append(results, ScanResult{
			Path:         path,
			Name:         d.Name(),
			IsDir:        d.IsDir(),
			RelativePath: relativePath,
		})

		return nil
	})

	return results, err
}

// ReadDir reads the contents of a directory and returns DirEntry slices
func (dm *DirectoryManager) ReadDir(dir string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	err := dm.withRoot(func(root *os.Root) error {
		fsys := root.FS()
		var err error
		entries, err = fs.ReadDir(fsys, dir)
		return err
	})

	return entries, err
}
