package main

import "net/http"

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	file, err := s.getFileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Path) {
		http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
		return
	}

	content, err := s.rootManager.ReadFile(file.Path)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}

	data := PageData{
		Title:         "Edit - " + file.Display,
		CurrentFile:   file,
		RawContent:    string(content),
		IsEditing:     true,
		CoreFiles:     s.getCoreFiles(file.Path),
		ResourceFiles: s.getResourceFiles(file.Path),
	}

	if err := s.executePage(w, "edit.html", data); err != nil {
		s.showServerError(w, r, err)
	}
}
