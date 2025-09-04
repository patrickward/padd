package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// ResolveMonthlyFile resolves the path for a monthly file based on the timestamp and file type.
func (dm *DirectoryManager) ResolveMonthlyFile(timestamp time.Time, fileType string) (string, error) {
	year := timestamp.Format("2006")
	month := timestamp.Format("01-January")

	dirPath := strings.ToLower(filepath.Join(fileType, year))
	filePath := strings.ToLower(filepath.Join(dirPath, month+".md"))

	// Ensure directory exists
	if err := dm.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
	}

	// Create the file if it doesn't exist
	if !dm.FileExists(filePath) {
		if err := dm.createMonthlyFile(filePath, timestamp); err != nil {
			return "", fmt.Errorf("failed to create dated file %s: %w", filePath, err)
		}
	}

	return filePath, nil
}

// createMonthlyFile creates a new monthly file with a header based on the timestamp
func (dm *DirectoryManager) createMonthlyFile(filePath string, timestamp time.Time) error {
	if dm.FileExists(filePath) {
		return nil
	}

	//content := fmt.Sprintf("# %s\n\n", timestamp.Format("January 2006"))
	return dm.WriteString(filePath, "\n")
}
