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

	directoryTree := s.fileRepo.DirectoryTreeFor(fileType)

	archiveFile := padd.FileInfo{
		ID:        fileType + "-archive",
		Path:      fileType + "/archive",
		Title:     padd.TitleCase(fileType) + " Archive",
		TitleBase: padd.TitleCase(fileType) + " Archive",
	}

	data := padd.PageData{
		Title:         archiveFile.Title,
		CurrentFile:   archiveFile,
		NavMenuFiles:  s.navigationMenu(fileType),
		ArchiveType:   fileType,
		DirectoryTree: directoryTree,
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
