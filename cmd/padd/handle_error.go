package main

import (
	"net/http"
)

func (s *Server) showPageNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := s.executePage(w, "404.html", PageData{
		Title:     "Page Not Found",
		CoreFiles: s.getCoreFiles(""),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) showServerError(w http.ResponseWriter, _ *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := s.executePage(w, "500.html", PageData{
		Title:        "Server Error",
		CoreFiles:    s.getCoreFiles(""),
		ErrorMessage: err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
