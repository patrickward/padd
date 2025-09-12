package main

import (
	"net/http"
	"strings"

	"github.com/patrickward/padd"
)

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
	years, files, err := s.fileRepo.TemporalTree(fileType)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	archiveFile := padd.FileInfo{
		ID:          fileType + "-archive",
		Path:        fileType + "/archive",
		Display:     padd.TitleCase(fileType) + " Archive",
		DisplayBase: padd.TitleCase(fileType) + " Archive",
	}

	data := padd.PageData{
		Title:         archiveFile.Display,
		CurrentFile:   archiveFile,
		NavMenuFiles:  s.navigationMenu(fileType),
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
