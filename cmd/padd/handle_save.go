package main

import "net/http"

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	doc, err := s.fileRepo.GetDocument(r.PathValue("id"))
	if err != nil {
		s.showPageNotFound(w, r)
		return
	}

	content := r.FormValue("content")
	if err = doc.Save(content); err != nil {
		s.showServerError(w, r, err)
		return
	}

	s.flashManager.SetSuccess(w, "File saved successfully")
	http.Redirect(w, r, "/"+doc.Info.ID, http.StatusSeeOther)
}
