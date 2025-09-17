package padd

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

const emptyFilePath = "untitled"

// FileRepository manages the core files and directories of the application.
type FileRepository struct {
	config            FileConfig
	rootManager       *RootManager
	cacheMux          sync.RWMutex
	lastCacheTime     time.Time
	directoryTree     *DirectoryNode
	fileIndex         map[string]FileInfo
	encryptionManager *EncryptionManager
}

// FileConfig holds the configuration for core files and directories.
type FileConfig struct {
	CoreFiles           []string
	ResourcesDirectory  string
	DailyDirectory      string
	JournalDirectory    string
	temporalDirectories []string
}

// TemporalDirectories returns the list of temporal directories (daily, journal).
func (fc FileConfig) TemporalDirectories() []string {
	return fc.temporalDirectories
}

// DefaultFileConfig provides default settings for FileRepository.
var DefaultFileConfig = FileConfig{
	CoreFiles:          []string{"inbox.md", "active.md"},
	ResourcesDirectory: "resources",
	DailyDirectory:     "daily",
	JournalDirectory:   "journal",
}

// NewFileRepository creates a new instance of FileRepository with the given configuration.
func NewFileRepository(rootManager *RootManager, config FileConfig) *FileRepository {
	config.temporalDirectories = []string{config.DailyDirectory, config.JournalDirectory}

	fr := &FileRepository{
		config:            config,
		rootManager:       rootManager,
		encryptionManager: NewEncryptionManager(),
	}

	return fr
}

// SetEncryptionManager sets the EncryptionManager for this FileRepository.
func (fr *FileRepository) SetEncryptionManager(manager *EncryptionManager) {
	fr.encryptionManager = manager
}

// EncryptionManager returns the EncryptionManager for this FileRepository.
func (fr *FileRepository) EncryptionManager() *EncryptionManager {
	return fr.encryptionManager
}

// Config returns the current FileConfig.
func (fr *FileRepository) Config() FileConfig {
	return fr.config
}

// Initialize sets up the core files and directories as per the configuration, ensuring they exist.
func (fr *FileRepository) Initialize() error {
	// Create the core files if they do not exist
	for _, file := range fr.config.CoreFiles {
		// Remove the md extension for CreateFileIfNotExists
		fileTitle := TitleCase(strings.TrimSuffix(file, ".md"))
		// Create the default frontmatter content
		frontmatter := "---\n" +
			"title: " + fileTitle + "\n" +
			"description: Your " + fileTitle + " file\n" +
			"---\n\n"
		err := fr.rootManager.CreateFileIfNotExists(file, frontmatter+"Enter your "+fileTitle+" here...")
		if err != nil {
			return fmt.Errorf("error creating core file %s: %v", file, err)
		}
	}

	// Create the resource directories if they do not exist
	if fr.config.ResourcesDirectory != "" {
		err := fr.rootManager.CreateDirectoryIfNotExists(fr.config.ResourcesDirectory)
		if err != nil {
			return fmt.Errorf("error creating resource directory %s: %v", fr.config.ResourcesDirectory, err)
		}
	}

	// Create the temporal directories if they do not exist
	for _, dir := range []string{fr.config.DailyDirectory, fr.config.JournalDirectory} {
		err := fr.rootManager.CreateDirectoryIfNotExists(dir)
		if err != nil {
			return fmt.Errorf("error creating temporal directory %s: %v", dir, err)
		}
	}

	return nil
}

// CoreFiles returns the cached list of core files.
func (fr *FileRepository) CoreFiles() map[string]FileInfo {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()
	//return fr.coreCache

	result := make(map[string]FileInfo)
	for _, coreFile := range fr.config.CoreFiles {
		id := strings.TrimSuffix(coreFile, ".md")
		if info, ok := fr.fileIndex[id]; ok {
			result[id] = info
		}
	}

	return result
}

// FileInfo retrieves the FileInfo for a given file id.
func (fr *FileRepository) FileInfo(id string) (FileInfo, error) {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()

	// Find the file in the fileIndex
	if info, ok := fr.fileIndex[id]; ok {
		return info, nil
	}

	// Look for a directory in the tree
	if fr.directoryTree != nil {
		if node := fr.directoryTree.FindDirectory(id); node != nil {
			display, displayBase := fr.DisplayName(id)
			return FileInfo{
				ID:            id,
				Path:          id,
				Title:         display,
				TitleBase:     displayBase,
				DirectoryPath: id,
				IsDirectory:   true,
				DirectoryNode: node,
				IsResource:    strings.HasPrefix(id, fr.config.ResourcesDirectory),
			}, nil
		}
	}

	return FileInfo{}, fmt.Errorf("file or directory %s not found", id)
}

// FileIsTemporal checks if a file with the given id is a temporal file (daily or journal).
func (fr *FileRepository) FileIsTemporal(id string) bool {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) > 0 && slices.Contains(fr.config.temporalDirectories, parts[0]) {
		return true
	}
	return false
}

// IsTemporalRoot checks if a file with the given id is a temporal root directory (daily or journal).
func (fr *FileRepository) IsTemporalRoot(id string) bool {
	return slices.Contains(fr.config.temporalDirectories, id)
}

// FileIDExists checks if a file with the given id exists in either core, resources, or temporal files.
func (fr *FileRepository) FileIDExists(id string) bool {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()

	_, exists := fr.fileIndex[id]
	return exists
}

// FilePathExists checks if a file with the given path exists in either core, resources, or temporal files.
func (fr *FileRepository) FilePathExists(path string) bool {
	return fr.rootManager.FileExists(path)
}

// ReloadCaches refreshes both the core files and resource caches.
func (fr *FileRepository) ReloadCaches() {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()

	tree, index := fr.buildDirectoryTree(".")
	fr.directoryTree = tree
	fr.fileIndex = index
	fr.lastCacheTime = time.Now()
	log.Printf("Cache refreshed with %d files", len(fr.fileIndex))
}

// printDirectoryTree prints the directory tree to a log.
func (fr *FileRepository) printDirectoryTree(tree *DirectoryNode, indent string) {
	for _, file := range tree.Files {
		log.Printf("%sFile: %s", indent, file.Path)
	}

	for _, dir := range tree.Directories {
		log.Printf("%sDirectory: %s", indent, dir.Name)
		fr.printDirectoryTree(dir, indent+"  ")
	}
}

// ReloadResources refreshes the resource files cache by rescanning the resources' directory.
func (fr *FileRepository) ReloadResources() {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()

	// If the directory tree is nil, there are no resources, so do nothing
	if fr.directoryTree == nil {
		return
	}

	// Otherwise, find the resource directory in the DirectoryNode tree if it exists
	_, ok := fr.directoryTree.Directories[fr.config.ResourcesDirectory]
	if !ok {
		log.Printf("Resource directory not found in tree, reloading all caches")
		//fr.ReloadCaches()
		// Create it
		fr.directoryTree.Directories[fr.config.ResourcesDirectory] = &DirectoryNode{
			Name:        fr.config.ResourcesDirectory,
			Files:       []FileInfo{},
			Directories: make(map[string]*DirectoryNode),
		}
	}

	// Now, build the directory for the resources directory
	tree, index := fr.buildDirectoryTree(fr.config.ResourcesDirectory)
	if tree == nil {
		log.Printf("Error building directory tree for resources directory, reloading all caches")
		fr.ReloadCaches()
		return
	}

	// When refreshing, we get the directory tree with the "resources" directory as the root.
	// So, we need to drill down to the resources directory and replace it with the new tree.
	if _, ok := tree.Directories[fr.config.ResourcesDirectory]; ok {
		fr.directoryTree.Directories[fr.config.ResourcesDirectory] = tree.Directories[fr.config.ResourcesDirectory]
	} else {
		log.Printf("Error replacing resources directory in directory tree, reloading all caches")
		fr.ReloadCaches()
		return
	}

	// Update or insert the resource files into the fileIndex
	for _, file := range index {
		fr.fileIndex[file.ID] = file
	}

	fr.lastCacheTime = time.Now()
	log.Printf("Resource cache refreshed with %d files", len(fr.fileIndex))
}

// ReloadResourcesIfStale refreshes the resource cache if it is older than the specified duration.
func (fr *FileRepository) ReloadResourcesIfStale(maxAge time.Duration) {
	fr.cacheMux.RLock()
	age := time.Since(fr.lastCacheTime)
	fr.cacheMux.RUnlock()

	if age > maxAge {
		fr.ReloadResources()
	}
}

// DirectoryTreeFor builds a hierarchical tree of resources based on their directory structure.
func (fr *FileRepository) DirectoryTreeFor(directory string) *DirectoryNode {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()

	emptyTree := &DirectoryNode{
		Name:        "",
		Files:       []FileInfo{},
		Directories: make(map[string]*DirectoryNode),
	}

	if fr.directoryTree == nil {
		log.Printf("Directory tree is nil, returning empty tree")
		return emptyTree
	}

	//if resourceNode, ok := fr.directoryTree.Directories[fr.config.ResourcesDirectory]; ok {
	if resourceNode, ok := fr.directoryTree.Directories[directory]; ok {
		return resourceNode
	}

	return emptyTree
}

// CreateID generates a consistent URL-safe ID from a file path
func (fr *FileRepository) CreateID(path string) string {
	if path == "" {
		return emptyFilePath
	}

	pathWithoutExt := strings.TrimSuffix(path, ".md")
	normalized := fr.normalizeFileName(pathWithoutExt)

	return normalized
}

// DisplayName generates a user-friendly display name from a file path
func (fr *FileRepository) DisplayName(relPath string) (string, string) {
	// Remove the "resources/" prefix and ".md" suffix
	pathWithoutPrefix := strings.TrimPrefix(relPath, fr.config.ResourcesDirectory+"/")
	pathWithoutSuffix := strings.TrimSuffix(pathWithoutPrefix, ".md")

	// Split into directory parts
	parts := strings.Split(pathWithoutSuffix, string(filepath.Separator))

	// Process each part: replace dashes/underscores with spaces and title case
	for i, part := range parts {
		part = strings.ReplaceAll(part, "-", " ")
		part = strings.ReplaceAll(part, "_", " ")
		parts[i] = TitleCase(part)
	}

	// Title is the full path with title-cased parts joined by "/"
	display := strings.Join(parts, "/")

	// TitleBase is just the last part (the file name without directory and without extension)
	displayBase := parts[len(parts)-1]
	return display, displayBase
}

func (fr *FileRepository) fileInfoFromPath(path string) FileInfo {
	id := fr.CreateID(path)

	// Extract directory info
	dir := filepath.Dir(path)
	if dir == "." {
		dir = "" // Root of resources
	}

	// Calculate depth
	depth := 0
	if dir != "" {
		depth = strings.Count(dir, string(filepath.Separator)) + 1
	}

	// Create a display name
	display, displayBase := fr.DisplayName(path)

	isResource := strings.HasPrefix(path, fr.config.ResourcesDirectory+"/")
	isTemporal := false

	for _, temporalDir := range fr.config.temporalDirectories {
		if strings.HasPrefix(path, temporalDir+"/") {
			isTemporal = true
			break
		}
	}

	fileInfo := FileInfo{
		ID:            id,
		Path:          path,
		Title:         display,
		TitleBase:     displayBase,
		DirectoryPath: dir,
		Depth:         depth,
		IsResource:    isResource,
		IsTemporal:    isTemporal,
	}

	return fileInfo
}

// normalizeFileName creates a URL-safe, consistent filename/path
// NOTE: This is obviously not perfect and could be improved for internationalization, etc.
// It's also not guaranteed to be unique, so collisions should be handled at a higher level if needed.
func (fr *FileRepository) normalizeFileName(path string) string {
	// Handle empty path
	if path == "" {
		return emptyFilePath
	}

	// Strip any .md extension
	path = strings.TrimSuffix(path, ".md")

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

	// Preallocate memory for the result
	result.Grow(len(normalized))

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

	// Clean up path separators - no hyphens immediately before or after
	cleaned = strings.ReplaceAll(cleaned, "-/", "/")
	cleaned = strings.ReplaceAll(cleaned, "/-", "/")
	cleaned = strings.ReplaceAll(cleaned, "-.", ".")
	cleaned = strings.ReplaceAll(cleaned, "./", "/")

	// Handle consecutive forward slashes
	cleaned = strings.ReplaceAll(cleaned, "//", "/")

	// Clean up leading and trailing hyphens
	cleaned = strings.Trim(cleaned, "-")

	// Replace multiple consecutive hyphens with single hyphen
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}

	cleaned = strings.Trim(cleaned, "-")
	cleaned = strings.Trim(cleaned, "/")
	cleaned = strings.TrimSpace(cleaned)

	// Handle edge case: if the path becomes empty after normalization
	if cleaned == "" {
		return emptyFilePath
	}

	return cleaned
}

// GetDocument retrieves a document by ID
func (fr *FileRepository) GetDocument(id string) (*Document, error) {
	info, err := fr.FileInfo(id)
	if err != nil {
		return nil, err
	}

	return &Document{
		Info: info,
		repo: fr,
	}, nil
}

// GetOrCreateResourceDocument retrieves a document by ID, or creates a new one if it doesn't exist.
// If the file doesn't exist, it will be created with default content. Files are always created
// in the ResourcesDirectory. You can omit the ResourcesDirectory prefix in the ID and it will be
// automatically added. Similarly, you can omit the .md extension and it will be added.
func (fr *FileRepository) GetOrCreateResourceDocument(id string) (*Document, error) {
	// Ensure the file is in the resources directory
	if !strings.HasPrefix(id, fr.Config().ResourcesDirectory+"/") {
		id = fr.Config().ResourcesDirectory + "/" + id
	}

	return fr.getOrCreateDocument(id)
}

// getOrCreateDocument gets or creates a document for a file, based on its ID.
// If the file doesn't exist, it will be created with default content.'
func (fr *FileRepository) getOrCreateDocument(id string) (*Document, error) {
	info, err := fr.FileInfo(id)
	if err == nil {
		return &Document{
			Info: info,
			repo: fr,
		}, nil
	}

	// File wasn't found, so create it
	path := id

	// Ensure .md extension
	if !strings.HasSuffix(path, ".md") {
		path += ".md"
	}

	// First, get the directory
	directory := filepath.Dir(path)
	if directory == "." {
		directory = ""
	}

	// Ensure the directory exists
	if err := fr.rootManager.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	// Create the file
	defaultContent := []byte("# " + filepath.Base(path) + "\n\n")
	if err := fr.rootManager.WriteFile(path, defaultContent, 0644); err != nil {
		return nil, fmt.Errorf("error creating file: %w", err)
	}

	// Reload the resources to include the new file
	fr.ReloadResources()

	// Get the file info again
	info, err = fr.FileInfo(id)
	if err != nil {
		return nil, fmt.Errorf("error getting file info: %w", err)
	}

	return &Document{
		Info: info,
		repo: fr,
	}, nil
}

// TemporalFileInfo retrieves or constructs a FileInfo for a temporal file based on type and date. If not
// found, the file will be created and returned.
func (fr *FileRepository) TemporalFileInfo(fileType string, timestamp time.Time) (FileInfo, bool) {
	year := timestamp.Format("2006")
	month := timestamp.Format("01-January")

	dirPath := strings.ToLower(filepath.Join(fileType, year))
	filePath := strings.ToLower(filepath.Join(dirPath, month+".md"))

	id := fr.CreateID(filePath)
	displayName := fmt.Sprintf("%s %d", timestamp.Format("January"), timestamp.Year())

	info := FileInfo{
		ID:            id,
		Path:          filePath,
		Title:         displayName,
		TitleBase:     displayName,
		IsTemporal:    true,
		DirectoryPath: fileType + "/" + timestamp.Format("2006"),
	}

	found := fr.rootManager.FileExists(filePath)

	return info, found
}

// GetOrCreateTemporalDocument gets or creates a document for a temporal file
func (fr *FileRepository) GetOrCreateTemporalDocument(directory string, date time.Time) (*Document, error) {
	info, found := fr.TemporalFileInfo(directory, date)

	if !found {
		// Make sure the directory exists
		dirPath := filepath.Dir(info.Path)
		if err := fr.rootManager.MkdirAll(dirPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}

		//content := fmt.Sprintf("# %s\n\n", timestamp.Format("January 2006"))
		err := fr.rootManager.WriteString(info.Path, "\n")
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %w", info.Path, err)
		}
	}

	return &Document{
		Info: info,
		repo: fr,
	}, nil
}

func (fr *FileRepository) DirectoryTree() *DirectoryNode {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()
	return fr.directoryTree
}

// buildDirectoryTree builds a directory tree from the root of the data directory. If the
// directory is empty, it will use the root of the data directory.
// It returns the root node and a map of all files in the tree, keyed by ID.
func (fr *FileRepository) buildDirectoryTree(directory string) (*DirectoryNode, map[string]FileInfo) {
	if directory == "" {
		directory = "."
	}

	root := &DirectoryNode{
		Name:        "",
		Files:       []FileInfo{},
		Directories: make(map[string]*DirectoryNode),
	}

	index := make(map[string]FileInfo)

	results, err := fr.rootManager.Scan(directory, func(path string, d fs.DirEntry) bool {
		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return false
		}

		// Skip hidden files and temp files
		if strings.HasPrefix(d.Name(), ".") || strings.HasPrefix(d.Name(), "~") {
			return false
		}

		return true
	})

	if err != nil {
		log.Printf("Error scanning resources directory: %v", err)
		return root, index
	}

	// Process each file and add to the tree and index
	for _, result := range results {
		fileInfo := fr.fileInfoFromPath(result.Path)
		fr.addFileToTree(root, fileInfo)
		index[fileInfo.ID] = fileInfo
	}

	return root, index
}

func (fr *FileRepository) addFileToTree(node *DirectoryNode, fileInfo FileInfo) {
	if fileInfo.DirectoryPath == "" {
		// File is at the root of the tree, so add it to the root node
		node.Files = append(node.Files, fileInfo)
		return
	}

	// Navigate directory structure and add file to the tree
	parts := strings.Split(fileInfo.DirectoryPath, string(filepath.Separator))
	currentNode := node

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

	currentNode.Files = append(currentNode.Files, fileInfo)
}
