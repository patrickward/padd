package main

import (
	"encoding/json"
	"net/http"

	"github.com/patrickward/padd/internal/files"
	"github.com/patrickward/padd/internal/web"
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
	if err := s.executePage(w, "404.html", web.PageData{
		Title:        "Page Not Found",
		NavMenuFiles: s.navigationMenu(""),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// isHXRequest returns true if the request header for HX-Request is true.
func isHXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// isHxSubmission returns true if the request is an HX-Request and is a POST, PUT, PATCH, or DELETE.
func isHxSubmission(r *http.Request) bool {
	if isHXRequest(r) {
		// Is the method post, put, patch, or delete?
		return r.Method == "POST" || r.Method == "PUT" || r.Method == "PATCH" || r.Method == "DELETE"
	}

	return false
}

// showServerError shows a 500 page response.
// If the request is an HX-Request, then send a 500 snippet response with the error message.
// Otherwise, show the 500 system error page.
func (s *Server) showServerError(w http.ResponseWriter, r *http.Request, err error) {
	if isHxSubmission(r) {
		// Send the system error snippet
		w.WriteHeader(http.StatusInternalServerError)
		if err := s.executeSnippet(w, "system_error.html", map[string]any{
			"ErrorMessage": err.Error(),
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Otherwise, send the generic 500 page
	w.WriteHeader(http.StatusInternalServerError)
	if err := s.executePage(w, "500.html", web.PageData{
		Title:        "Server Error",
		NavMenuFiles: s.navigationMenu(""),
		ErrorMessage: err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) respondWithJSONError(w http.ResponseWriter, payload any, code int) {
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) showDocumentError(w http.ResponseWriter, r *http.Request, doc *files.Document, err error) {
	w.WriteHeader(http.StatusInternalServerError)

	w.WriteHeader(http.StatusInternalServerError)
	if err := s.executePage(w, "500.html", web.PageData{
		Title:        "Server Error",
		CurrentFile:  doc.Info,
		NavMenuFiles: s.navigationMenu(""),
		ErrorMessage: err.Error(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
