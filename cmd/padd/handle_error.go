package main

import (
	"net/http"

	"github.com/patrickward/padd"
)

func (s *Server) showPageNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := s.executePage(w, "404.html", padd.PageData{
		Title:        "Page Not Found",
		NavMenuFiles: s.navigationMenu(""),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) showServerError(w http.ResponseWriter, _ *http.Request, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	if err := s.executePage(w, "500.html", padd.PageData{
		Title:        "Server Error",
		NavMenuFiles: s.navigationMenu(""),
		ErrorMessage: err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
