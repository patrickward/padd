package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

const (
	appName      = "PADD"
	appVersion   = "0.1.0"
	resourcesDir = "resources"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	dataDir       string
	dirManager    *DirectoryManager
	md            goldmark.Markdown
	baseTempl     *template.Template // Common templates (layouts, partials)
	resourceCache []FileInfo
	cacheMux      sync.RWMutex
	lastCacheTime time.Time
	flashManager  *FlashManager
}

type FileInfo struct {
	ID          string
	Name        string
	Display     string
	DisplayBase string // Base name without directory
	IsCurrent   bool
	Directory   string // Directory path relative to the resources/ (empty for core and files at the root of resources/)
	Depth       int    // Depth in the resources/ directory structure (0 for core and files at the root of resources/)
}

type DirectoryNode struct {
	Name        string
	Files       []FileInfo
	Directories map[string]*DirectoryNode
}

type PageData struct {
	Title            string
	CurrentFile      FileInfo
	Content          template.HTML
	RawContent       string
	IsEditing        bool
	IsSearching      bool
	IsResources      bool
	CoreFiles        []FileInfo
	ResourceFiles    []FileInfo
	ResourceTree     *DirectoryNode
	SearchQuery      string
	SearchResults    map[string][]SearchMatch
	FlashMessage     string
	FlashMessageType string
	ErrorMessage     string
	SearchMatch      int // To indicate which match in the line to highlight
}

type SearchMatch struct {
	LineNum    int           // The line number in the file (1-based)
	Line       string        // The raw line text
	Rendered   template.HTML // The rendered HTML of the line (for display)
	MatchIndex int           // The index of the match in the line, for potential highlighting
}

var filesSort = []string{"inbox", "active", "daily"}

var filesMap = map[string]FileInfo{
	"inbox":  {ID: "inbox", Name: "inbox.md", Display: "Inbox", DisplayBase: "Inbox"},
	"active": {ID: "active", Name: "active.md", Display: "Active", DisplayBase: "Active"},
	"daily":  {ID: "daily", Name: "daily.md", Display: "Daily Log", DisplayBase: "Daily Log"},
}

func NewServer(dataDir string) (*Server, error) {
	dirManager, err := NewDirectoryManager(dataDir)
	if err != nil {
		return nil, err
	}

	// Initialize markdown parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Typographer,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithAttribute(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(), // Allow raw HTML
		),
	)

	// Parse templates with custom functions
	funcMap := template.FuncMap{
		"contains": strings.Contains,
		"toLower":  strings.ToLower,
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict requires an even number of arguments")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layouts/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		return nil, err
	}

	s := &Server{
		dataDir:      dataDir,
		dirManager:   dirManager,
		md:           md,
		baseTempl:    tmpl,
		flashManager: NewFlashManager(),
	}

	s.initializeFiles()
	s.refreshResourceCache()
	go s.backgroundCacheRefresh()

	return s, nil
}

func (s *Server) executePage(w http.ResponseWriter, page string, data PageData) error {
	// Clone the base template to avoid altering it
	tmpl, err := s.baseTempl.Clone()
	if err != nil {
		return err
	}

	// Add .html extension if missing
	if !strings.HasSuffix(page, ".html") {
		page = page + ".html"
	}

	// Parse the specific page template
	pagePattern := fmt.Sprintf("templates/pages/%s", page)
	tmpl, err = tmpl.ParseFS(templateFS, pagePattern)
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(w, page, data)
}

func (s *Server) refreshResourceCache() {
	s.cacheMux.Lock()
	defer s.cacheMux.Unlock()

	s.resourceCache = s.scanResourceFiles("")
	s.lastCacheTime = time.Now()
	//log.Printf("Resource cache refreshed with %d files", len(s.resourceCache))
}

func (s *Server) backgroundCacheRefresh() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		// Only refresh if cache is older than 1 minute
		s.cacheMux.RLock()
		shouldRefresh := time.Since(s.lastCacheTime) > time.Minute
		s.cacheMux.RUnlock()

		if shouldRefresh {
			s.refreshResourceCache()
		}
	}
}

func (s *Server) initializeFiles() {
	defaults := map[string]string{
		"inbox.md":  "Capture everything here first.\n\n",
		"active.md": "Active projects, links, and tasks.\n\n",
		"daily.md":  "Daily activities and logs.\n\n",
	}

	for file, content := range defaults {
		if err := s.dirManager.CreateFileIfNotExists(file, content); err != nil {
			log.Printf("Error creating default file %s: %v", file, err)
			continue
		}
	}
}

func (s *Server) getCoreFiles(current string) []FileInfo {
	var files []FileInfo

	for _, id := range filesSort {
		if f, ok := filesMap[id]; ok {
			fileCopy := f
			fileCopy.IsCurrent = fileCopy.Name == current
			files = append(files, fileCopy)
		}
	}

	return files
}

func (s *Server) getResourceFiles(current string) []FileInfo {
	s.cacheMux.RLock()
	defer s.cacheMux.RUnlock()

	// Return a copy to avoid race conditions
	filesCopy := make([]FileInfo, len(s.resourceCache))
	copy(filesCopy, s.resourceCache)
	return filesCopy
}

func (s *Server) buildDirectoryTree(files []FileInfo) *DirectoryNode {
	root := &DirectoryNode{
		Name:        "",
		Files:       []FileInfo{},
		Directories: make(map[string]*DirectoryNode),
	}

	for _, file := range files {
		if file.Directory == "" {
			// File is at the root of resources/
			root.Files = append(root.Files, file)
			continue
		}

		parts := strings.Split(file.Directory, string(filepath.Separator))
		currentNode := root

		for _, part := range parts {
			if _, exists := currentNode.Directories[part]; !exists {
				currentNode.Directories[part] = &DirectoryNode{
					Name:        part,
					Files:       []FileInfo{},
					Directories: make(map[string]*DirectoryNode),
				}
			}
			currentNode = currentNode.Directories[part]
		}

		currentNode.Files = append(currentNode.Files, file)
	}

	return root
}

func (s *Server) scanResourceFiles(current string) []FileInfo {
	// Create the resources directory if it doesn't exist
	if err := s.dirManager.MkdirAll(resourcesDir, 0755); err != nil {
		log.Printf("Error creating resources directory: %v", err)
		return []FileInfo{}
	}

	results, err := s.dirManager.Scan(resourcesDir, func(path string, d fs.DirEntry) bool {
		return !d.IsDir() && strings.HasSuffix(d.Name(), ".md")
	})

	if err != nil {
		log.Printf("Error scanning resources directory: %v", err)
		return []FileInfo{}
	}

	var files []FileInfo
	for _, result := range results {
		// Create ID from relative path (replace separators)
		//id := strings.ReplaceAll(result.Path, string(filepath.Separator), "_")
		//id = strings.TrimSuffix(id, ".md")
		id := s.createID(result.Path)

		// Extract directory info
		pathWithoutPrefix := strings.TrimPrefix(result.Path, resourcesDir+"/")
		dir := filepath.Dir(pathWithoutPrefix)
		if dir == "." {
			dir = "" // Root of resources
		}

		// Calculate depth
		depth := 0
		if dir != "" {
			depth = strings.Count(dir, string(filepath.Separator)) + 1
		}

		// Create display name
		display := s.createDisplayName(result.Path)
		displayBase := strings.TrimSuffix(filepath.Base(result.Name), ".md")
		displayBase = strings.ReplaceAll(displayBase, "-", " ")
		displayBase = strings.ReplaceAll(displayBase, "_", " ")
		//goland:noinspection GoDeprecation
		displayBase = strings.Title(displayBase)

		files = append(files, FileInfo{
			ID:          id,
			Name:        result.Path,
			Display:     display,
			DisplayBase: displayBase,
			IsCurrent:   result.Path == current,
			Directory:   dir,
			Depth:       depth,
		})
	}

	// Sort files alphabetically by display name for consistency
	sort.Slice(files, func(i, j int) bool {
		// Primary sort: Root files (empty directory) should come before any directory files
		// This ensures all root-level files appear at the top, regardless of name

		// Return true if i should come before j
		if files[i].Directory == "" && files[j].Directory != "" {
			return true
		}

		// Returning false here means j should come before i
		if files[i].Directory != "" && files[j].Directory == "" {
			return false
		}

		// Secondary sort: By directory name
		if files[i].Directory != files[j].Directory {
			return files[i].Directory < files[j].Directory
		}

		// Tertiary sort: By display name
		return files[i].Display < files[j].Display
	})

	return files
}

// normalizeFileName creates a URL-safe, consistent filename/path
func (s *Server) normalizeFileName(path string) string {
	// Convert to lowercase for consistency
	normalized := strings.ToLower(path)

	// Always use forward slashes for URLs
	normalized = strings.ReplaceAll(normalized, string(filepath.Separator), "/")

	// Replace spaces and underscores with hyphens
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")

	// Remove or replace other problematic characters
	// Keep only: letters, numbers, hyphens, periods, and forward slashes
	var result strings.Builder
	for _, char := range normalized {
		switch {
		case (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'):
			result.WriteRune(char)
		case char == '-' || char == '.' || char == '/':
			result.WriteRune(char)
		default:
			// Replace other characters with hyphens, but avoid consecutive hyphens
			if result.Len() > 0 && result.String()[result.Len()-1] != '-' {
				result.WriteRune('-')
			}
		}
	}

	// Clean up any trailing hyphens or multiple consecutive hyphens
	cleaned := result.String()
	cleaned = strings.Trim(cleaned, "-")

	// Replace multiple consecutive hyphens with single hyphen
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}

	return cleaned
}

// createID generates a consistent URL-safe ID from a file path
func (s *Server) createID(path string) string {
	// Remove the .md extension and normalize
	pathWithoutExt := strings.TrimSuffix(path, ".md")
	normalized := s.normalizeFileName(pathWithoutExt)

	//Remove resources/ prefix if present
	//normalized = strings.TrimPrefix(normalized, "resources/")

	return normalized
}

func (s *Server) createDisplayName(relPath string) string {
	// Remove the "resources/" prefix and ".md" suffix
	pathWithoutPrefix := strings.TrimPrefix(relPath, resourcesDir+"/")
	pathWithoutSuffix := strings.TrimSuffix(pathWithoutPrefix, ".md")

	// Split into directory parts
	parts := strings.Split(pathWithoutSuffix, string(filepath.Separator))

	// Process each part: replace dashes/underscores with spaces and title case
	for i, part := range parts {
		part = strings.ReplaceAll(part, "-", " ")
		part = strings.ReplaceAll(part, "_", " ")
		//goland:noinspection GoDeprecation
		parts[i] = strings.Title(part)
	}

	// Join with "/" to show hierarchy
	return strings.Join(parts, "/")
}

func (s *Server) isValidFile(fileName string) bool {
	coreFiles := []string{"inbox.md", "active.md", "daily.md"}
	for _, valid := range coreFiles {
		if fileName == valid {
			return true
		}
	}

	if strings.HasPrefix(fileName, resourcesDir+"/") && strings.HasSuffix(fileName, ".md") {
		return s.dirManager.Exists(fileName)
	}

	return false
}

func (s *Server) renderMarkdown(content string) template.HTML {
	contentWithShortcodes := s.processShortcodes(content)

	var buf bytes.Buffer
	if err := s.md.Convert([]byte(contentWithShortcodes), &buf); err != nil {
		return template.HTML(fmt.Sprintf("<pre>%s</pre>", template.HTMLEscapeString(contentWithShortcodes)))
	}

	// Process inline svg images to ensure they are displayed correctly
	processedContent := s.processInlineSVG(buf.String())

	return template.HTML(processedContent)
}

func (s *Server) renderMarkdownWithHighlight(content, query string, targetIndex int) template.HTML {
	if query == "" || targetIndex < 1 {
		return s.renderMarkdown(content)
	}

	contentWithShortcodes := s.processShortcodes(content)

	lines := strings.Split(contentWithShortcodes, "\n")
	queryLower := strings.ToLower(query)
	matchIndex := 1

	// Process each line and add match IDs
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			// Check if the line starts with list markers
			trimmed := strings.TrimLeft(line, " \t")
			var listMarker, content string

			if strings.HasPrefix(trimmed, "- ") {
				prefixLen := len(line) - len(trimmed) + 2 // account for "- "
				listMarker = line[:prefixLen]
				content = line[prefixLen:]
			} else if strings.HasPrefix(trimmed, "* ") {
				prefixLen := len(line) - len(trimmed) + 2 // account for "* "
				listMarker = line[:prefixLen]
				content = line[prefixLen:]
			} else {
				listMarker = ""
				content = line
			}

			// Apply highlighting to the content part only
			if matchIndex == targetIndex {
				// Add an ID to the line for scrolling
				lines[i] = listMarker + fmt.Sprintf(`<span id="search-match-%d" class="search-highlight search-target">%s</span>`, matchIndex, content)
			} else {
				lines[i] = listMarker + fmt.Sprintf(`<span id="search-match-%d" class="search-highlight">%s</span>`, matchIndex, content)
			}

			matchIndex++
		}
	}

	modifiedContent := strings.Join(lines, "\n")
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(modifiedContent), &buf); err != nil {
		return template.HTML(fmt.Sprintf("<pre>%s</pre>", template.HTMLEscapeString(content)))
	}

	// Process inline svg images to ensure they are displayed correctly
	processedContent := s.processInlineSVG(buf.String())
	return template.HTML(processedContent)
}

func stripHeaders(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	// Strip header markers (anything starting with #, ##, ###, or ####)
	trimmed = strings.TrimLeft(trimmed, "#")
	trimmed = strings.TrimLeft(trimmed, " \t")
	return trimmed
}

// stripMarkers removes leading list markers (-, *) and whitespace from a line
func stripMarkers(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") {
		return stripHeaders(strings.TrimPrefix(trimmed, "- "))
	}
	if strings.HasPrefix(trimmed, "* ") {
		return stripHeaders(strings.TrimPrefix(trimmed, "* "))
	}

	return stripHeaders(trimmed)
}

func (s *Server) getFileInfo(id string) (FileInfo, error) {
	if file, ok := filesMap[id]; ok {
		return file, nil
	}

	// Check resource files
	resourceFiles := s.getResourceFiles(id)
	for _, file := range resourceFiles {
		if file.ID == id {
			return file, nil
		}
	}

	if id == "" {
		return filesMap["inbox"], nil
	}

	return FileInfo{}, fmt.Errorf("file with ID %s not found", id)
}

func (s *Server) searchFile(file FileInfo, query string) []SearchMatch {
	var matches []SearchMatch
	content, err := s.dirManager.ReadFile(file.Name)
	if err != nil {
		return matches
	}

	lines := strings.Split(string(content), "\n")
	matchIndex := 1 // To track the occurrence of matches in a line
	queryLower := strings.ToLower(query)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {

			cleanedLine := stripMarkers(line)
			matches = append(matches, SearchMatch{
				LineNum:    i + 1,
				Line:       line,
				Rendered:   s.renderMarkdown(cleanedLine),
				MatchIndex: matchIndex,
			})
			matchIndex++
		}
	}

	return matches
}

// Handlers

// Main function

// getDataDirectory determines the data directory using a tiered approach:
// 1. Command-line flag (-data) takes highest precedence.
// 2. Environment variable PADD_DATA_DIR if flag is not set.
// 3. XDG_DATA_HOME/padd or $HOME/.local/share/padd as fallback.
func getDataDirectory(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}

	if envDir := os.Getenv("PADD_DATA_DIR"); envDir != "" {
		return envDir, nil
	}

	xdgDataHome := os.Getenv("XDG_DATA_HOME")
	if xdgDataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("unable to determine user home directory: %v", err)
		}
		xdgDataHome = filepath.Join(homeDir, ".local", "share")
	}

	return filepath.Join(xdgDataHome, "padd"), nil
}

func main() {
	var port int
	var addr string
	var dataFlag string
	var showVersion bool

	// Note to self about Flag aliases: Go's flag package allows multiple flag names to point to the same variable.
	// When you call BoolVar/StringVar/etc. multiple times with the same variable pointer,
	// you create aliases that all modify the same memory location. This enables both short
	// and long flag versions (e.g., -v and -version) without needing separate variables.
	// The default value is only applied once, not overridden - both flags share the same
	// default and will set the same variable when used by the user.
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagSet.StringVar(&dataFlag, "data", "", "Directory to store markdown files.")
	flagSet.StringVar(&dataFlag, "d", "", "Directory to store markdown files.")
	flagSet.IntVar(&port, "port", 8080, "Port to run the server on.")
	flagSet.IntVar(&port, "p", 8080, "Port to run the server on.")
	flagSet.StringVar(&addr, "addr", "localhost", "Address to bind the server to.")
	flagSet.StringVar(&addr, "a", "localhost", "Address to bind the server to.")
	flagSet.BoolVar(&showVersion, "version", false, "Show application version.")
	flagSet.BoolVar(&showVersion, "v", false, "Show application version.")

	flagSet.Usage = func() {
		_, _ = fmt.Fprintf(flagSet.Output(), "PADD - Personal Assistant for Daily Documentation\n\n")
		flagSet.PrintDefaults()
	}

	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(fmt.Errorf("error parsing flags: %v", err))
	}

	if showVersion {
		fmt.Printf("PADD version %s\n", appVersion)
		os.Exit(0)
		return
	}

	resolvedDataDir, err := getDataDirectory(dataFlag)
	if err != nil {
		log.Fatal(fmt.Errorf("error determining data directory: %v", err))
	}

	server, err := NewServer(resolvedDataDir)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing server: %v", err))
	}

	serverAddr := fmt.Sprintf("%s:%d", addr, port)

	mux := http.NewServeMux()

	// Serve static files
	fileServer := http.FileServer(http.FS(staticFS))
	mux.Handle("GET /static/", fileServer)

	// Serve images (both embedded defaults and user-provided)
	mux.Handle("GET /images/", server.handleImages())

	// API routes
	mux.HandleFunc("GET /api/icons", server.handleIconsAPI)
	mux.HandleFunc("GET /api/resources", server.handleResourcesAPI)

	// Routes using new Go 1.22+ patterns
	//mux.HandleFunc("GET /", server.handleView)
	mux.HandleFunc("GET /edit/{id...}", server.handleEdit)
	mux.HandleFunc("POST /save/{id...}", server.handleSave)
	mux.HandleFunc("POST /daily", server.handleDaily)
	mux.HandleFunc("POST /inbox/add", server.handleInboxAdd)
	mux.HandleFunc("GET /search", server.handleSearch)
	mux.HandleFunc("GET /resources", server.handleResources)
	mux.HandleFunc("POST /resources/create", server.handleCreateResource)
	mux.HandleFunc("POST /admin/refresh", server.handleRefreshCache)
	mux.HandleFunc("GET /{id...}", server.handleView)

	fmt.Printf("Server starting on https://%s\n", serverAddr)
	fmt.Printf("Data directory: %s\n", resolvedDataDir)
	log.Fatal(http.ListenAndServe(serverAddr, mux))
}
