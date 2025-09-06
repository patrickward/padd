package main

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"

	"github.com/patrickward/padd"
)

// Server holds the application state and configuration
type Server struct {
	dataDir          string
	rootManager      *padd.RootManager
	fileRepo         *padd.FileRepository
	flashManager     *padd.FlashManager
	backgroundRunner *padd.BackgroundRunner
	md               goldmark.Markdown
	baseTempl        *template.Template // Common templates (layouts, partials)
	httpServer       *http.Server
	sanitizer        *bluemonday.Policy
	metadataConfig   MetadataConfig
}

// NewServer initializes the server with the given data directory
func NewServer(ctx context.Context, dataDir string) (*Server, error) {
	rootManager, err := padd.NewRootManager(dataDir)
	if err != nil {
		return nil, err
	}

	fileRepo := padd.NewFileRepository(rootManager, padd.DefaultFileConfig)

	md := createMarkdownRenderer(rootManager)
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	// Initialize background task runner
	backgroundRunner := padd.NewBackgroundRunner(ctx)

	s := &Server{
		dataDir:          dataDir,
		rootManager:      rootManager,
		fileRepo:         fileRepo,
		md:               md,
		baseTempl:        tmpl,
		flashManager:     padd.NewFlashManager(),
		backgroundRunner: backgroundRunner,
		sanitizer:        createSanitizer(),
	}

	err = s.fileRepo.Initialize()
	if err != nil {
		return nil, fmt.Errorf("could not initialize file repository: %w", err)
	}

	s.setupMetadataConfig()
	s.fileRepo.ReloadCaches()
	s.setupBackgroundTasks()

	return s, nil
}

func createSanitizer() *bluemonday.Policy {
	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowAttrs("class", "id").OnElements("span", "div", "i", "code", "pre", "p", "h1", "h2", "h3", "h4", "h5", "h6")

	// Allow form elements, so we can use them in markdown for checklists, etc.
	sanitizer.AllowElements("form", "input", "textarea", "button", "select", "option", "label")
	sanitizer.AllowAttrs("type", "checked", "disabled", "name", "value", "placeholder").OnElements("input", "textarea", "button", "select", "option", "label")

	// Allow all of the "hx-*" attributes for htmx (https://htmx.org/)
	sanitizer.AllowAttrs("hx-get", "hx-post", "hx-put", "hx-delete", "hx-patch", "hx-target", "hx-swap", "hx-trigger", "hx-vals", "hx-include", "hx-headers", "hx-push-url", "hx-confirm", "hx-indicator", "hx-params").
		OnElements("a", "form", "button", "input", "select", "textarea", "div", "span", "p")

	// Allow media elements
	// "audio" "svg" "video" are all permitted
	sanitizer.AllowElements("audio", "svg", "video")
	sanitizer.AllowAttrs("autoplay", "controls", "loop", "muted", "preload", "src", "type", "width", "height").OnElements("audio", "video")
	sanitizer.AllowAttrs("xmlns", "viewbox", "width", "height", "fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin").OnElements("svg", "path", "circle", "rect", "line", "polyline", "polygon")
	sanitizer.AllowAttrs("d", "cx", "cy", "r", "x", "y", "x1", "y1", "x2", "y2", "points").OnElements("path", "circle", "rect", "line", "polyline", "polygon")
	return sanitizer
}

func (s *Server) setupBackgroundTasks() {
	// TODO: Make the cache duration configurable
	backgroundCacheDuration := 5 * time.Minute

	s.backgroundRunner.AddPeriodicTask(
		"cache-refresh",
		backgroundCacheDuration,
		func(ctx context.Context) error {
			s.fileRepo.ReloadResourcesIfStale(backgroundCacheDuration)
			return nil
		},
	)

	// Example: Add other background tasks as needed
	// s.backgroundRunner.AddPeriodicTask(
	//     "health-check",
	//     5*time.Minute,
	//     func(ctx context.Context) error {
	//         // Perform health checks
	//         return nil
	//     },
	// )
}

// backgroundTask wraps a function to run as a one-time background task with panic recovery
//
// Example usage:
//
//	s.backgroundTask("send-email", func() error {
//	    // Task logic here
//	    return nil
//	})
func (s *Server) backgroundTask(name string, fn func() error) {
	s.backgroundRunner.StartOneTimeTask(name, func(ctx context.Context) error {
		defer func() {
			if r := recover(); r != nil {
				// Handle panic
				fmt.Printf("Background task %s panicked: %v\n", name, r)
			}
		}()

		return fn()
	})
}

// Start starts the server and all background tasks
func (s *Server) Start(add string, port int) error {
	serverAddr := fmt.Sprintf("%s:%d", add, port)

	s.httpServer = &http.Server{
		Addr:         serverAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  time.Minute,
		Handler:      s.setupRoutes(),
	}

	// Start background tasks
	s.backgroundRunner.Start()

	// Channel to receive OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the http server in a separate goroutine
	serverErrors := make(chan error, 1)
	go func() {
		fmt.Printf("Starting server on %s\n", serverAddr)
		fmt.Printf("Data directory: %s\n", s.dataDir)
		serverErrors <- s.httpServer.ListenAndServe()
	}()

	// Wait for either termination signal or server error
	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("could not start server: %w", err)
		}
	case sig := <-sigChan:
		fmt.Printf("Received signal %v, initiating shutdown\n", sig)
	}

	return s.Shutdown()
}

// Shutdown gracefully shuts down the server and background tasks
func (s *Server) Shutdown() error {
	log.Println("Shutting down server...")

	// Create a timeout context for the shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown the HTTP server
	if s.httpServer != nil {
		log.Println("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error during HTTP server shutdown: %v", err)
		}
	}

	// Shutdown background tasks
	s.backgroundRunner.Shutdown()

	log.Println("Server shutdown complete")
	return nil
}

// navigationMenu returns the list of navigation menu items
// TODO: Make this configurable
func (s *Server) navigationMenu(current string) []padd.FileInfo {

	current = strings.TrimPrefix(current, "/")
	if current == "" {
		current = "inbox"
	}

	files := []padd.FileInfo{
		{
			ID:          "inbox",
			Path:        "inbox.md",
			Display:     "Inbox",
			DisplayBase: "Inbox",
			IsNavActive: current == "inbox",
			IsCurrent:   current == "inbox",
		},
		{
			ID:          "active",
			Path:        "active.md",
			Display:     "Active",
			DisplayBase: "Active",
			IsNavActive: current == "active",
			IsCurrent:   current == "active",
		},
		{
			ID:          "daily",
			Path:        "daily",
			Display:     "Daily",
			DisplayBase: "Daily",
			IsTemporal:  true,
			IsNavActive: current == "daily" || strings.HasPrefix(current, "daily/"),
			IsCurrent:   current == "daily" || strings.HasPrefix(current, "daily/"),
		},
		{
			ID:          "journal",
			Path:        "journal",
			Display:     "Journal",
			DisplayBase: "Journal",
			IsTemporal:  true,
			IsNavActive: current == "journal" || strings.HasPrefix(current, "journal/"),
			IsCurrent:   current == "journal" || strings.HasPrefix(current, "journal/"),
		},
		{
			ID:          "resources",
			Path:        "resources",
			Display:     "Resources",
			DisplayBase: "Resources",
			IsResource:  true,
			IsNavActive: current == "resources" || strings.HasPrefix(current, "resources/"),
			IsCurrent:   current == "resources" || strings.HasPrefix(current, "resources/"),
		},
	}

	return files
}
