package padd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RootManager provides safe filesystem operations within a specific directory using os.Root
type RootManager struct {
	path string
}

// NewRootManager creates a new RootManager for the given directory path
func NewRootManager(path string) (*RootManager, error) {
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

	return &RootManager{path: path}, nil
}

// withRoot executes a function with a safely opened os.Root
func (rm *RootManager) withRoot(fn func(*os.Root) error) error {
	root, err := os.OpenRoot(rm.path)
	if err != nil {
		return fmt.Errorf("failed to open root: %w", err)
	}
	defer func(root *os.Root) {
		_ = root.Close()
	}(root)

	return fn(root)
}

// ReadFile reads the contents of a file using Root.ReadFile
func (rm *RootManager) ReadFile(filename string) ([]byte, error) {
	var content []byte
	err := rm.withRoot(func(root *os.Root) error {
		var err error
		content, err = root.ReadFile(filename)
		return err
	})
	return content, err
}

// WriteFile writes content to a file using Root.WriteFile
func (rm *RootManager) WriteFile(filename string, content []byte, perm os.FileMode) error {
	return rm.withRoot(func(root *os.Root) error {
		return root.WriteFile(filename, content, perm)
	})
}

// WriteString writes a string to a file
func (rm *RootManager) WriteString(filename string, content string) error {
	return rm.WriteFile(filename, []byte(content), 0644)
}

// FileExists checks if a file exists using Root.Stat
func (rm *RootManager) FileExists(filename string) bool {
	exists := false
	_ = rm.withRoot(func(root *os.Root) error {
		_, err := root.Stat(filename)
		exists = err == nil
		return nil
	})
	return exists
}

// Stat returns file info using Root.Stat
func (rm *RootManager) Stat(filename string) (os.FileInfo, error) {
	var info os.FileInfo
	err := rm.withRoot(func(root *os.Root) error {
		var err error
		info, err = root.Stat(filename)
		return err
	})
	return info, err
}

// MkdirAll creates a directory and any necessary parent directories using Root.MkdirAll
func (rm *RootManager) MkdirAll(dir string, perm os.FileMode) error {
	return rm.withRoot(func(root *os.Root) error {
		return root.MkdirAll(dir, perm)
	})
}

// Remove removes a file using Root.Remove
func (rm *RootManager) Remove(filename string) error {
	return rm.withRoot(func(root *os.Root) error {
		return root.Remove(filename)
	})
}

// RemoveAll removes a directory and all its contents using Root.RemoveAll
func (rm *RootManager) RemoveAll(path string) error {
	return rm.withRoot(func(root *os.Root) error {
		return root.RemoveAll(path)
	})
}

// WalkDir walks the directory tree using Root.FS()
func (rm *RootManager) WalkDir(root string, fn fs.WalkDirFunc) error {
	return rm.withRoot(func(osRoot *os.Root) error {
		fsys := osRoot.FS()
		return fs.WalkDir(fsys, root, fn)
	})
}

// CreateFileIfNotExists creates a file with default content if it doesn't exist
func (rm *RootManager) CreateFileIfNotExists(filename string, defaultContent string) error {
	if rm.FileExists(filename) {
		return nil
	}
	return rm.WriteString(filename, defaultContent)
}

// CreateDirectoryIfNotExists creates a directory if it doesn't exist
func (rm *RootManager) CreateDirectoryIfNotExists(dir string) error {
	info, err := rm.Stat(dir)
	if os.IsNotExist(err) {
		return rm.MkdirAll(dir, 0755)
	}
	if err != nil {
		return fmt.Errorf("failed to check directory %s: %w", dir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s exists but is not a directory", dir)
	}

	return nil
}

// ScanResult holds information about a scanned file or directory
type ScanResult struct {
	Path         string
	Name         string
	IsDir        bool
	RelativePath string
}

// Scan scans the directory tree starting from rootDir, applying an optional filter function
func (rm *RootManager) Scan(rootDir string, filter func(string, fs.DirEntry) bool) ([]ScanResult, error) {
	var results []ScanResult

	err := rm.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
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
func (rm *RootManager) ReadDir(dir string) ([]fs.DirEntry, error) {
	var entries []fs.DirEntry
	err := rm.withRoot(func(root *os.Root) error {
		fsys := root.FS()
		var err error
		entries, err = fs.ReadDir(fsys, dir)
		return err
	})

	return entries, err
}

// ResolveMonthlyFile resolves the path for a monthly file based on the timestamp and file type. If not
// found, it will create the file.
func (rm *RootManager) ResolveMonthlyFile(timestamp time.Time, fileType string) (string, error) {
	year := timestamp.Format("2006")
	month := timestamp.Format("01-January")

	dirPath := strings.ToLower(filepath.Join(fileType, year))
	filePath := strings.ToLower(filepath.Join(dirPath, month+".md"))

	// Ensure directory exists
	if err := rm.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Create the file if it doesn't exist
	if !rm.FileExists(filePath) {
		if err := rm.createMonthlyFile(filePath, timestamp); err != nil {
			return "", fmt.Errorf("failed to create dated file %s: %w", filePath, err)
		}
	}

	return filePath, nil
}

// createMonthlyFile creates a new monthly file with a header based on the timestamp
func (rm *RootManager) createMonthlyFile(filePath string, timestamp time.Time) error {
	if rm.FileExists(filePath) {
		return nil
	}

	// Make sure the directory exists
	dirPath := filepath.Dir(filePath)
	if err := rm.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	//content := fmt.Sprintf("# %s\n\n", timestamp.Format("January 2006"))
	return rm.WriteString(filePath, "\n")
}
