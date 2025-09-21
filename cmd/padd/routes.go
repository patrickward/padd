package main

import (
	"net/http"

	"github.com/patrickward/padd"
)

func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Serve static files
	fileServer := http.FileServer(http.FS(padd.StaticFS))
	mux.Handle("GET /static/", fileServer)

	// Serve images (both embedded defaults and user-provided)
	mux.Handle("GET /images/", s.handleImages())
	mux.HandleFunc("GET /api/icons", s.handleIconsAPI)
	mux.HandleFunc("POST /api/images/upload", s.handleImageUpload)

	// Tasks
	mux.HandleFunc("PATCH /tasks/toggle/{id...}", s.handleTaskToggle)
	mux.HandleFunc("GET /tasks/edit/{id...}", s.handleTaskEdit)
	mux.HandleFunc("GET /tasks/show/{id...}", s.handleTaskShow)
	mux.HandleFunc("POST /tasks/complete/{id...}", s.handleArchiveDoneTasks)
	mux.HandleFunc("PATCH /tasks/{id...}", s.handleTaskUpdate)
	mux.HandleFunc("DELETE /tasks/{id...}", s.handleTaskDelete)

	// Content
	mux.HandleFunc("GET /edit/{id...}", s.handleEdit)
	mux.HandleFunc("GET /daily/archive", s.handleTemporalArchive)
	mux.HandleFunc("GET /daily", s.handleTemporalRoot("daily"))
	mux.HandleFunc("POST /daily", s.handleAddTemporalEntry("daily"))
	mux.HandleFunc("POST /add/{id...}", s.handleAddEntry)
	mux.HandleFunc("GET /journal/archive", s.handleTemporalArchive)
	mux.HandleFunc("GET /journal", s.handleTemporalRoot("journal"))
	mux.HandleFunc("POST /journal", s.handleAddTemporalEntry("journal"))
	mux.HandleFunc("GET /search", s.handleSearch)
	mux.HandleFunc("GET /resources", s.handleResources)
	mux.HandleFunc("POST /resources", s.handleCreateResource)
	mux.HandleFunc("POST /resources/refresh", s.handleRefreshResources)
	mux.HandleFunc("GET /page-header/{id...}", s.handlePageHeader)
	mux.HandleFunc("POST /{id...}", s.handleSave)

	// Handles page views and root
	mux.HandleFunc("GET /{id...}", s.handleView)

	return mux
}
