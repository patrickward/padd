package main

import (
	"net/http"

	"github.com/patrickward/padd"
)

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	doc, err := s.fileRepo.GetDocument(id)
	if err != nil {
		s.showServerError(w, r, err)
	}

	content, err := doc.Content()
	if err != nil {
		s.showServerError(w, r, err)
	}

	data := padd.PageData{
		Title:        "Edit - " + doc.Info.Display,
		CurrentFile:  doc.Info,
		RawContent:   content,
		IsEditing:    true,
		NavMenuFiles: s.navigationMenu(id),
	}

	if err := s.executePage(w, "edit.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}
