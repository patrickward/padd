package main

import "net/http"

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	file, err := s.getFileInfo(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	if !s.isValidFile(file.Path) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")
	if err := s.dirManager.WriteString(file.Path, content); err != nil {
		s.showServerError(w, r, err)
		return
	}

	s.flashManager.SetSuccess(w, "File saved successfully")
	http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
}
