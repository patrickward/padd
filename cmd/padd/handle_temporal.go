package main

import (
	"net/http"
	"strings"
)

func (s *Server) isTemporalFile(filePath string) bool {
	return strings.HasPrefix(filePath, "daily/") || strings.HasPrefix(filePath, "journal/")
}

func (s *Server) handleTemporalArchive(w http.ResponseWriter, r *http.Request) {
	// Get the type from the URL path (e.g., /daily/archive is "daily", /journal/archive is "journal")
	path := strings.Trim(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || (parts[0] != "daily" && parts[0] != "journal") || parts[1] != "archive" {
		s.showPageNotFound(w, r)
		return
	}

	fileType := parts[0]

	// Get all available temporal files of the specified type
	years, files, err := s.getTemporalFiles(fileType)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	archiveFile := FileInfo{
		ID:          fileType + "-archive",
		Path:        fileType + "/archive",
		Display:     strings.Title(fileType) + " Archive",
		DisplayBase: strings.Title(fileType) + " Archive",
		IsCurrent:   false,
	}

	data := PageData{
		Title:         archiveFile.Display,
		CurrentFile:   archiveFile,
		CoreFiles:     s.getCoreFiles(fileType),
		ResourceFiles: s.getResourceFiles(fileType),
		TemporalYears: years,
		TemporalFiles: files,
		ArchiveType:   fileType,
	}

	// Check for flash messages
	if flash := s.flashManager.Get(w, r); flash != nil {
		data.FlashMessage = flash.Message
		data.FlashMessageType = flash.Type
	}

	if err := s.executePage(w, "temporal_archive.html", data); err != nil {
		s.showServerError(w, r, err)
		return
	}
}
