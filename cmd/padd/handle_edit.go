package main

import (
	"net/http"

	"github.com/patrickward/padd"
)

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	file, err := s.fileRepo.FileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.fileRepo.FilePathExists(file.Path) {
		http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
		return
	}

	content, err := s.rootManager.ReadFile(file.Path)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	data := padd.PageData{
		Title:        "Edit - " + file.Display,
		CurrentFile:  file,
		RawContent:   string(content),
		IsEditing:    true,
		NavMenuFiles: s.navigationMenu(file.ID),
	}

	if err := s.executePage(w, "edit.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}
