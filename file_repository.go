package padd

import (
	"fmt"
	"io/fs"
	"log"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileRepository manages the core files and directories of the application.
type FileRepository struct {
	config        FileConfig
	rootManager   *RootManager
	cacheMux      sync.RWMutex
	lastCacheTime time.Time
	coreCache     map[string]FileInfo
	resourceCache map[string]FileInfo
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
		config:        config,
		rootManager:   rootManager,
		coreCache:     make(map[string]FileInfo),
		resourceCache: make(map[string]FileInfo),
	}

	fr.ReloadCoreFiles()

	return fr
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
	return fr.coreCache
}

// ResourceFiles returns the cached list of resource files.
func (fr *FileRepository) ResourceFiles() map[string]FileInfo {
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()
	return fr.resourceCache
}

// FileInfo retrieves the FileInfo for a given file id.
func (fr *FileRepository) FileInfo(id string) (FileInfo, error) {
	// If the id is equal to one of the temporal directories, show the current date file for that directory
	parts := strings.SplitN(id, "/", 2)

	// Check if it's a temporal directory
	if len(parts) > 0 && slices.Contains(fr.config.temporalDirectories, parts[0]) {
		filePath := id + ".md"
		// ids are like daily/2025/09-september, so the filePath is daily/2025/09-september.md
		// and the display name is "September 2025"
		// We can construct the display name from the parts
		if len(parts) < 2 {
			return FileInfo{}, fmt.Errorf("invalid temporal file id: %s", id)
		}

		// parts[1] is like 2025/09-september
		subParts := strings.SplitN(parts[1], "/", 2)
		if len(subParts) < 2 {
			return FileInfo{}, fmt.Errorf("invalid temporal file id: %s", id)
		}

		// subParts[0] is the year, subParts[1] is like 09-september
		monthParts := strings.SplitN(subParts[1], "-", 2)
		if len(monthParts) < 2 {
			return FileInfo{}, fmt.Errorf("invalid temporal file id: %s", id)
		}

		monthNumber := monthParts[0]
		monthName := monthParts[1]
		displayName := fmt.Sprintf("%s %s", TitleCase(monthName), subParts[0])

		if fr.rootManager.FileExists(filePath) {
			return FileInfo{
				ID:          id,
				Path:        filePath,
				Display:     displayName,
				DisplayBase: displayName,
				Directory:   parts[0] + "/" + subParts[0],
				Year:        subParts[0],
				Month:       monthNumber,
				MonthName:   TitleCase(monthName),
				IsTemporal:  true,
			}, nil
		}
	}

	// Lock for reading the caches
	fr.cacheMux.RLock()
	defer fr.cacheMux.RUnlock()

	// Check core files
	if info, ok := fr.CoreFiles()[id]; ok {
		return info, nil
	}

	// Check resources
	if info, ok := fr.resourceCache[id]; ok {
		return info, nil
	}

	return FileInfo{}, fmt.Errorf("file %s not found", id)
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

	if _, ok := fr.CoreFiles()[id]; ok {
		return true
	}

	if _, ok := fr.resourceCache[id]; ok {
		return true
	}

	// Check if it's a temporal directory
	if fr.FileIsTemporal(id) {
		return fr.rootManager.FileExists(id)
	}

	return false
}

// FilePathExists checks if a file with the given path exists in either core, resources, or temporal files.
func (fr *FileRepository) FilePathExists(path string) bool {
	return fr.rootManager.FileExists(path)
}

// ReloadCaches refreshes both the core files and resource caches.
func (fr *FileRepository) ReloadCaches() {
	fr.ReloadCoreFiles()
	fr.ReloadResources()
}

// ReloadResources refreshes the resource files cache by rescanning the resources' directory.
func (fr *FileRepository) ReloadResources() {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()
	fr.resourceCache = fr.scanResources()
	fr.lastCacheTime = time.Now()
	log.Printf("Resource cache refreshed with %d files", len(fr.resourceCache))
}

// ReloadResource reloads a single resource file.
func (fr *FileRepository) ReloadResource(path string) {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()

	if fr.rootManager.FileExists(path) {
		fr.resourceCache[fr.CreateID(path)] = fr.fileInfoFromPath(path)
	}
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

// ResourcesTree builds a hierarchical tree of resources based on their directory structure.
func (fr *FileRepository) ResourcesTree() *DirectoryNode {
	files := fr.sortedResources()

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

// CreateID generates a consistent URL-safe ID from a file path
func (fr *FileRepository) CreateID(path string) string {
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

	// Display is the full path with title-cased parts joined by "/"
	display := strings.Join(parts, "/")

	// DisplayBase is just the last part (the file name without directory and without extension)
	displayBase := parts[len(parts)-1]
	return display, displayBase
}

// ReloadCoreFiles refreshes the core files cache.
func (fr *FileRepository) ReloadCoreFiles() {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()

	coreFiles := make(map[string]FileInfo, len(fr.config.CoreFiles))
	for _, file := range fr.config.CoreFiles {
		if fr.rootManager.FileExists(file) {
			name := strings.TrimSuffix(file, ".md")
			title := TitleCase(name)
			coreFiles[name] = FileInfo{
				ID:          name,
				Path:        file,
				Display:     title,
				DisplayBase: title,
			}
		}
	}

	fr.coreCache = coreFiles
}

// TemporalTree builds a hierarchical tree of temporal files (daily, journal) based on their directory structure.
func (fr *FileRepository) TemporalTree(fileType string) (years []string, files map[string][]FileInfo, err error) {
	files = make(map[string][]FileInfo)

	// Check if the directory exists
	yearEntries, err := fr.rootManager.ReadDir(fileType)
	if err != nil {
		return []string{}, files, nil // Return empty list if directory doesn't exist
	}

	for _, yearEntry := range yearEntries {
		if !yearEntry.IsDir() {
			continue
		}

		yearPath := filepath.Join(fileType, yearEntry.Name())
		monthEntries, err := fr.rootManager.ReadDir(yearPath)
		if err != nil {
			continue // Skip this year if there's an error
		}

		// Create the year entry if it doesn't exist
		if _, exists := files[yearEntry.Name()]; !exists {
			files[yearEntry.Name()] = []FileInfo{}
		}

		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() && strings.HasSuffix(monthEntry.Name(), ".md") {
				monthName := strings.TrimSuffix(monthEntry.Name(), ".md")
				filePath := filepath.Join(yearPath, monthEntry.Name())
				id := fmt.Sprintf("%s/%s/%s", fileType, yearEntry.Name(), monthName)

				parts := strings.SplitN(monthName, "-", 2)
				displayName := monthName // Fallback to raw month name
				monthNumber := parts[0]
				monthDisplay := monthName
				if len(parts) == 2 {
					displayName = fmt.Sprintf("%s %s", TitleCase(parts[1]), yearEntry.Name())
					monthDisplay = TitleCase(parts[1])
				}

				files[yearEntry.Name()] = append(files[yearEntry.Name()], FileInfo{
					ID:          id,
					Path:        filePath,
					Display:     displayName,
					DisplayBase: displayName,
					Directory:   fileType + "/" + yearEntry.Name(),
					Year:        yearEntry.Name(),
					Month:       monthNumber,
					MonthName:   monthDisplay,
				})
			}
		}

		// Sort months within the year
		sort.Slice(files[yearEntry.Name()], func(i, j int) bool {
			return files[yearEntry.Name()][i].Month > files[yearEntry.Name()][j].Month // Reverse chronological order
		})
	}

	years = slices.Sorted(maps.Keys(files))
	slices.Reverse(years)

	return years, files, nil
}

// scanResources scans the resources directory for markdown files and builds the resource cache.
func (fr *FileRepository) scanResources() map[string]FileInfo {
	// Create the resources directory if it doesn't exist
	if err := fr.rootManager.MkdirAll(fr.config.ResourcesDirectory, 0755); err != nil {
		log.Printf("Error creating resources directory: %v", err)
		return map[string]FileInfo{}
	}

	results, err := fr.rootManager.Scan(fr.config.ResourcesDirectory, func(path string, d fs.DirEntry) bool {
		return !d.IsDir() && strings.HasSuffix(d.Name(), ".md")
	})

	if err != nil {
		log.Printf("Error scanning resources directory: %v", err)
		return map[string]FileInfo{}
	}

	var files = make(map[string]FileInfo, len(results))

	for _, result := range results {
		fileInfo := fr.fileInfoFromPath(result.Path)
		files[fileInfo.ID] = fileInfo
	}

	return files
}

func (fr *FileRepository) fileInfoFromPath(path string) FileInfo {
	id := fr.CreateID(path)

	// Extract directory info
	pathWithoutPrefix := strings.TrimPrefix(path, fr.config.ResourcesDirectory+"/")
	dir := filepath.Dir(pathWithoutPrefix)
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

	return FileInfo{
		ID:          id,
		Path:        path,
		Display:     display,
		DisplayBase: displayBase,
		Directory:   dir,
		Depth:       depth,
		IsResource:  true,
	}
}

// sortedResources returns a slice of FileInfo sorted by directory and display name.
func (fr *FileRepository) sortedResources() []FileInfo {
	fr.cacheMux.Lock()
	defer fr.cacheMux.Unlock()

	resources := maps.Values(fr.resourceCache)

	files := slices.SortedFunc(resources, func(a, b FileInfo) int {
		// Primary sort: Root files (empty directory) should come before any directory files
		// This ensures all root-level files appear at the top, regardless of name
		if a.Directory == "" && b.Directory != "" {
			return -1 // a comes before b
		}

		if a.Directory != "" && b.Directory == "" {
			return 1 // b comes before a
		}

		// Secondary sort: By directory name
		if a.Directory != b.Directory {
			return strings.Compare(a.Directory, b.Directory)
		}

		// Tertiary sort: By display name
		return strings.Compare(a.Display, b.Display)
	})

	return files
}

// normalizeFileName creates a URL-safe, consistent filename/path
// NOTE: This is obviously not perfect and could be improved for internationalization, etc.
// It's also not guaranteed to be unique, so collisions should be handled at a higher level if needed.
func (fr *FileRepository) normalizeFileName(path string) string {
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

	// Reload the cache for this single file
	fr.ReloadResource(path)

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
	//filePath, err := fr.rootManager.ResolveMonthlyFile(date, fileType)
	//if err != nil {
	//	return FileInfo{}, err
	//}
	//
	//id := fr.CreateID(filePath)
	//displayName := fmt.Sprintf("%s %d", date.Format("January"), date.Year())
	//
	//return FileInfo{
	//	ID:          id,
	//	Path:        filePath,
	//	Display:     displayName,
	//	DisplayBase: displayName,
	//	IsTemporal:  true,
	//	Directory:   fileType + "/" + date.Format("2006"),
	//	Year:        date.Format("2006"),
	//	Month:       date.Format("01"),
	//	MonthName:   date.Format("January"),
	//}, nil

	year := timestamp.Format("2006")
	month := timestamp.Format("01-January")

	dirPath := strings.ToLower(filepath.Join(fileType, year))
	filePath := strings.ToLower(filepath.Join(dirPath, month+".md"))

	id := fr.CreateID(filePath)
	displayName := fmt.Sprintf("%s %d", timestamp.Format("January"), timestamp.Year())

	info := FileInfo{
		ID:          id,
		Path:        filePath,
		Display:     displayName,
		DisplayBase: displayName,
		IsTemporal:  true,
		Directory:   fileType + "/" + timestamp.Format("2006"),
		Year:        timestamp.Format("2006"),
		Month:       timestamp.Format("01"),
		MonthName:   timestamp.Format("January"),
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

//// findTemporalFile resolves the path for a monthly file based on the timestamp and file type. If not
//// found, it will create the file.
//func (fr *FileRepository) findTemporalFile(fileType string, timestamp time.Time) (string, error) {
//	year := timestamp.Format("2006")
//	month := timestamp.Format("01-January")
//
//	dirPath := strings.ToLower(filepath.Join(fileType, year))
//	filePath := strings.ToLower(filepath.Join(dirPath, month+".md"))
//
//	// Ensure directory exists
//	if err := fr.rootManager.MkdirAll(dirPath, 0755); err != nil {
//		return "", fmt.Errorf("failed to create directory %s: %w", dirPath, err)
//	}
//
//	// Create the file if it doesn't exist
//	if !fr.rootManager.FileExists(filePath) {
//		if err := fr.createMonthlyFile(filePath, timestamp); err != nil {
//			return "", fmt.Errorf("failed to create dated file %s: %w", filePath, err)
//		}
//	}
//
//	return filePath, nil
//}

//// createMonthlyFile creates a new monthly file with a header based on the timestamp
//func (fr *FileRepository) createMonthlyFile(filePath string, timestamp time.Time) error {
//	if fr.rootManager.FileExists(filePath) {
//		return nil
//	}
//
//	// Make sure the directory exists
//	dirPath := filepath.Dir(filePath)
//	if err := fr.rootManager.MkdirAll(dirPath, 0755); err != nil {
//		return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
//	}
//
//	//content := fmt.Sprintf("# %s\n\n", timestamp.Format("January 2006"))
//	return fr.rootManager.WriteString(filePath, "\n")
//}
