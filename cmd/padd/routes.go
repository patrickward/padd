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

	// API routes
	mux.HandleFunc("GET /api/icons", s.handleIconsAPI)
	mux.HandleFunc("GET /api/resources", s.handleResourcesAPI)
	mux.HandleFunc("PATCH /api/tasks/toggle/{id...}", s.handleTaskToggle)
	mux.HandleFunc("GET /api/tasks/edit/{id...}", s.handleTaskEdit)
	mux.HandleFunc("GET /api/tasks/show/{id...}", s.handleTaskShow)
	mux.HandleFunc("POST /api/tasks/complete/{id...}", s.handleArchiveDoneTasks)
	mux.HandleFunc("PATCH /api/tasks/{id...}", s.handleTaskUpdate)
	mux.HandleFunc("DELETE /api/tasks/{id...}", s.handleTaskDelete)

	mux.HandleFunc("GET /edit/{id...}", s.handleEdit)
	mux.HandleFunc("GET /daily/archive", s.handleTemporalArchive)
	mux.HandleFunc("POST /daily", s.handleAddTemporalEntry("daily"))
	mux.HandleFunc("POST /add/{id...}", s.handleAddEntry)
	mux.HandleFunc("GET /journal/archive", s.handleTemporalArchive)
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
