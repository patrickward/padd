package main

import (
	"net/http"

	"github.com/patrickward/padd"
)

// redirectTo redirects the request to the given URL based on the request headers.
// If the request header for HX-Request is true, then send a 204 with a HX-Redirect header.
// Otherwise, send a 302 redirect.
func (s *Server) redirectTo(w http.ResponseWriter, r *http.Request, url string) {
	// If the request header for HX-Request is true, then send a 204 with a HX-Redirect header
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}

// showPageNotFound shows a 404 page.
func (s *Server) showPageNotFound(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := s.executePage(w, "404.html", padd.PageData{
		Title:        "Page Not Found",
		NavMenuFiles: s.navigationMenu(""),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// showServerError shows a 500 page.
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
