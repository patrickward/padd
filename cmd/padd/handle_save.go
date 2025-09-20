package main

import (
	"log"
	"net/http"
)

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	doc, err := s.fileRepo.GetDocument(r.PathValue("id"))
	if err != nil {
		log.Println("!! Failed to get file:", err)
		s.showPageNotFound(w, r)
		return
	}

	log.Println("!! Found doc")

	content := r.FormValue("content")
	if err = doc.Save(content); err != nil {
		log.Println("!! Failed to save file:", err)
		s.showServerError(w, r, err)
		return
	}

	log.Println("!! Saved file")
	s.flashManager.SetSuccess(w, "File saved successfully")
	s.redirectTo(w, r, "/"+doc.Info.ID)
}
