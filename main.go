package main

import (
	"bytes"
	"embed"
	"flag"
	"fmt"
	"html/template"
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

const appName = "PADD"

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	dataDir       string
	md            goldmark.Markdown
	baseTempl     *template.Template // Common templates (layouts, partials)
	resourceCache []FileInfo
	cacheMux      sync.RWMutex
	lastCacheTime time.Time
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
	Title         string
	CurrentFile   FileInfo
	Content       template.HTML
	RawContent    string
	IsEditing     bool
	IsSearching   bool
	IsResources   bool
	CanEdit       bool
	CoreFiles     []FileInfo
	ResourceFiles []FileInfo
	ResourceTree  *DirectoryNode
	SearchQuery   string
	SearchResults map[string][]SearchMatch
	Message       string
	MessageType   string
	SearchMatch   int // To indicate which match in the line to highlight
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
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	// Initialize markdown parser with extensions
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.TaskList,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
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
		"replace":  strings.ReplaceAll,
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templateFS,
		"templates/layouts/*.html",
		"templates/partials/*.html",
	)
	if err != nil {
		return nil, err
	}

	s := &Server{
		dataDir:   dataDir,
		md:        md,
		baseTempl: tmpl,
	}

	// Initialize files if they don't exist
	s.initializeFiles()
	s.refreshResourceCache() // Initial cache population
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
	log.Printf("Resource cache refreshed with %d files", len(s.resourceCache))
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
		path := filepath.Join(s.dataDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = os.WriteFile(path, []byte(content), 0644)
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
	var files []FileInfo
	resourceDir := filepath.Join(s.dataDir, "resources")

	// Create the resources directory if it doesn't exist
	if err := os.MkdirAll(resourceDir, 0755); err != nil {
		return files
	}

	// Walk through the resources directory and list markdown files
	err := filepath.Walk(resourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking despite errors for now
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, err := filepath.Rel(s.dataDir, path)
			if err != nil {
				return nil
			}

			// Create ID from relative path (replace separators)
			id := strings.ReplaceAll(relPath, string(filepath.Separator), "_")
			id = strings.TrimSuffix(id, ".md")

			// Extract directory info
			pathWithoutPrefix := strings.TrimPrefix(relPath, "resources/")
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
			display := s.createDisplayName(relPath)
			displayBase := strings.TrimSuffix(info.Name(), ".md")
			displayBase = strings.ReplaceAll(displayBase, "-", " ")
			displayBase = strings.ReplaceAll(displayBase, "_", " ")
			//goland:noinspection GoDeprecation
			displayBase = strings.Title(displayBase)

			files = append(files, FileInfo{
				ID:          id,
				Name:        relPath,
				Display:     display,
				DisplayBase: displayBase,
				IsCurrent:   relPath == current,
				Directory:   dir,
				Depth:       depth,
			})
		}
		return nil
	})

	if err != nil {
		log.Printf("Error walking resource directory: %v", err)
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

func (s *Server) createDisplayName(relPath string) string {
	// Remove the "resources/" prefix and ".md" suffix
	pathWithoutPrefix := strings.TrimPrefix(relPath, "resources/")
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

	if strings.HasPrefix(fileName, "resources/") && strings.HasSuffix(fileName, ".md") {
		fullPath := filepath.Join(s.dataDir, fileName)
		if _, err := os.Stat(fullPath); err == nil {
			return true
		}
	}

	return false
}

func (s *Server) renderMarkdown(content string) template.HTML {
	var buf bytes.Buffer
	if err := s.md.Convert([]byte(content), &buf); err != nil {
		return template.HTML(fmt.Sprintf("<pre>%s</pre>", template.HTMLEscapeString(content)))
	}
	return template.HTML(buf.String())
}

func (s *Server) renderMarkdownWithHighlight(content, query string, targetIndex int) template.HTML {
	if query == "" || targetIndex < 1 {
		return s.renderMarkdown(content)
	}

	lines := strings.Split(content, "\n")
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
	return template.HTML(buf.String())
}

// stripListMarkers removes leading list markers (-, *) and whitespace from a line
func stripListMarkers(line string) string {
	trimmed := strings.TrimLeft(line, " \t")
	if strings.HasPrefix(trimmed, "- ") {
		return strings.TrimPrefix(trimmed, "- ")
	}
	if strings.HasPrefix(trimmed, "* ") {
		return strings.TrimPrefix(trimmed, "* ")
	}
	return line
}

func (s *Server) getFileInfo(id string) FileInfo {
	if file, ok := filesMap[id]; ok {
		return file
	}

	// Check resource files
	resourceFiles := s.getResourceFiles(id)
	for _, file := range resourceFiles {
		if file.ID == id {
			return file
		}
	}

	return filesMap["inbox"]
}

func (s *Server) searchFile(file FileInfo, query string) []SearchMatch {
	var matches []SearchMatch
	content, err := os.ReadFile(filepath.Join(s.dataDir, file.Name))
	if err != nil {
		return matches
	}

	lines := strings.Split(string(content), "\n")
	matchIndex := 1 // To track the occurrence of matches in a line
	queryLower := strings.ToLower(query)
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), queryLower) {
			cleanedLine := stripListMarkers(line)
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

func (s *Server) handleView(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return
	}

	content, err := os.ReadFile(filepath.Join(s.dataDir, file.Name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Get search query and match parameters
	searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	var searchMatch int
	if matchStr := r.URL.Query().Get("match"); matchStr != "" {
		_, _ = fmt.Sscanf(matchStr, "%d", &searchMatch)
	}

	// Render content with search highlighting if needed
	var renderedContent template.HTML
	if searchQuery != "" {
		renderedContent = s.renderMarkdownWithHighlight(string(content), searchQuery, searchMatch)
	} else {
		renderedContent = s.renderMarkdown(string(content))
	}

	data := PageData{
		Title:         file.Display + " - " + appName,
		CurrentFile:   file,
		Content:       renderedContent,
		RawContent:    string(content),
		CoreFiles:     s.getCoreFiles(file.Name),
		ResourceFiles: s.getResourceFiles(file.Name),
		CanEdit:       file.Name != "daily.md",
		SearchQuery:   searchQuery,
		SearchMatch:   searchMatch,
	}

	// Check for message in query params (after redirect from save/daily)
	if msg := r.URL.Query().Get("msg"); msg != "" {
		data.Message = msg
		data.MessageType = r.URL.Query().Get("type")
		if data.MessageType == "" {
			data.MessageType = "success"
		}
	}

	if err := s.executePage(w, "view.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleRefreshCache(w http.ResponseWriter, r *http.Request) {
	s.refreshResourceCache()
	http.Redirect(w, r, "/resources", http.StatusSeeOther)
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) || file.Name == "daily.md" {
		http.Redirect(w, r, "/"+file.ID, http.StatusSeeOther)
		return
	}

	content, err := os.ReadFile(filepath.Join(s.dataDir, file.Name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := PageData{
		Title:         "Edit - " + file.Display + " - " + appName,
		CurrentFile:   file,
		RawContent:    string(content),
		IsEditing:     true,
		CoreFiles:     s.getCoreFiles(file.Name),
		ResourceFiles: s.getResourceFiles(file.Name),
	}

	if err := s.executePage(w, "edit.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
	file := s.getFileInfo(r.PathValue("id"))

	if !s.isValidFile(file.Name) || file.Name == "daily.md" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	content := r.FormValue("content")
	filePath := filepath.Join(s.dataDir, file.Name)

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/"+file.ID+"?msg=File saved successfully&type=success", http.StatusSeeOther)
}

func (s *Server) handleDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/daily", http.StatusSeeOther)
		return
	}

	entry := strings.TrimSpace(r.FormValue("entry"))
	if entry == "" {
		http.Redirect(w, r, "/daily?msg=Entry cannot be empty&type=danger", http.StatusSeeOther)
		return
	}

	// Read existing daily file
	dailyPath := filepath.Join(s.dataDir, "daily.md")
	existingContent, err := os.ReadFile(dailyPath)
	if err != nil {
		existingContent = []byte("# Daily Log\n\n")
	}

	// Format new entry with seconds in timestamp
	now := time.Now()
	dateHeader := fmt.Sprintf("## %s", now.Format("2006-01-02"))
	timeStamp := now.Format("15:04:05")
	newEntry := fmt.Sprintf("- `%s` %s", timeStamp, entry)

	// Parse existing content
	lines := strings.Split(string(existingContent), "\n")
	var result []string
	dateFound := false

	for _, line := range lines {
		if line == dateHeader {
			dateFound = true
			result = append(result, line)
			// Insert the new entry immediately after the date header
			result = append(result, newEntry)
		} else {
			result = append(result, line)
		}
	}

	// If date header wasn't found, add it at the top (after main header)
	if !dateFound {
		// Find where to insert (after the main header and any blank lines)
		insertPos := 0
		for i, line := range lines {
			if strings.HasPrefix(line, "# ") {
				insertPos = i + 1
				// Skip blank lines after header
				for insertPos < len(lines) && strings.TrimSpace(lines[insertPos]) == "" {
					insertPos++
				}
				break
			}
		}

		// Insert the new section
		result = nil
		result = append(result, lines[:insertPos]...)
		result = append(result, dateHeader)
		result = append(result, newEntry)
		result = append(result, "") // blank line after section
		if insertPos < len(lines) {
			result = append(result, lines[insertPos:]...)
		}
	}

	// Write back to file
	updatedContent := strings.Join(result, "\n")
	if err := os.WriteFile(dailyPath, []byte(updatedContent), 0644); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := fmt.Sprintf("Entry added at %s", now.Format("15:04:05"))
	http.Redirect(w, r, "/daily?msg="+msg+"&type=success", http.StatusSeeOther)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	results := make(map[string][]SearchMatch)

	// Search core files
	for _, file := range filesMap {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	// Search resource files
	resourceFiles := s.getResourceFiles("")
	for _, file := range resourceFiles {
		if matches := s.searchFile(file, query); len(matches) > 0 {
			results[file.ID] = matches
		}
	}

	data := PageData{
		Title:         "Search Results - " + appName,
		IsSearching:   true,
		SearchQuery:   query,
		SearchResults: results,
		CoreFiles:     s.getCoreFiles(""),
		ResourceFiles: s.getResourceFiles(""),
	}

	if err := s.executePage(w, "search.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleResources shows a list of available resource files
func (s *Server) handleResources(w http.ResponseWriter, r *http.Request) {
	resourceFiles := s.getResourceFiles("")
	resourceTree := s.buildDirectoryTree(resourceFiles)

	data := PageData{
		Title:         "Resources - " + appName,
		CoreFiles:     s.getCoreFiles(""),
		IsResources:   true,
		ResourceFiles: resourceFiles,
		ResourceTree:  resourceTree,
	}

	if err := s.executePage(w, "resources.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&dataFlag, "data", "", "Directory to store markdown files.")
	fs.IntVar(&port, "port", 8080, "Port to run the server on.")
	fs.StringVar(&addr, "addr", "localhost", "Address to bind the server to.")

	fs.Usage = func() {
		_, _ = fmt.Fprintf(fs.Output(), "PADD - Personal Assistant for Daily Documentation\n\n")
		fs.PrintDefaults()
	}

	err := fs.Parse(os.Args[1:])
	if err != nil {
		log.Fatal(fmt.Errorf("error parsing flags: %v", err))
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

	// Routes using new Go 1.22+ patterns
	mux.HandleFunc("GET /", server.handleView)
	mux.HandleFunc("GET /edit/{id}", server.handleEdit)
	mux.HandleFunc("POST /save/{id}", server.handleSave)
	mux.HandleFunc("POST /daily", server.handleDaily)
	mux.HandleFunc("GET /search", server.handleSearch)
	mux.HandleFunc("GET /resources", server.handleResources)
	mux.HandleFunc("GET /{id}", server.handleView)
	mux.HandleFunc("POST /admin/refresh", server.handleRefreshCache)

	fmt.Printf("Server starting on https://%s\n", serverAddr)
	fmt.Printf("Data directory: %s\n", resolvedDataDir)
	log.Fatal(http.ListenAndServe(serverAddr, mux))
}
