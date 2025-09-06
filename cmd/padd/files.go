package main

//// coreFiles are the essential markdown files always present in the root. Note that daily and journal are not
//// actual files but temporal files created as needed. They are listed here for navigation purposes.
//var coreFilesSort = []string{"inbox", "active", "daily", "journal"}
//
//// coreFilesMap maps core file IDs to their metadata. These files are always present. Note that daily and journal are
//// not actual files but temporal files created as needed. They are listed here for navigation purposes.
//var coreFilesMap = map[string]FileInfo{
//	"inbox":   {ID: "inbox", Path: "inbox.md", Display: "Inbox", DisplayBase: "Inbox"},
//	"active":  {ID: "active", Path: "active.md", Display: "Active", DisplayBase: "Active"},
//	"daily":   {ID: "daily", Path: "daily", Display: "Daily Log", DisplayBase: "Daily Log", IsTemporal: true},
//	"journal": {ID: "journal", Path: "journal", Display: "Journal", DisplayBase: "Journal", IsTemporal: true},
//}

//// refreshResourceCache scans the resources directory and updates the in-memory cache
//func (s *Server) refreshResourceCache() {
//	s.cacheMux.Lock()
//	defer s.cacheMux.Unlock()
//	s.resourceCache = s.scanResourceFiles("")
//	s.lastCacheTime = time.Now()
//	log.Printf("Resource cache refreshed with %d files", len(s.resourceCache))
//}

//// initializeFiles ensures core markdown files exist with default content
//func (s *Server) initializeFiles() {
//	defaults := map[string]string{
//		"inbox.md":  "# Inbox\n\nCapture everything here first.\n\n",
//		"active.md": "# Active\n\nActive projects, links, and tasks.\n\n",
//	}
//
//	for file, content := range defaults {
//		if err := s.rootManager.CreateFileIfNotExists(file, content); err != nil {
//			log.Printf("Error creating default file %s: %v", file, err)
//			continue
//		}
//	}
//}

//// getCoreFiles returns the list of core files with the current file marked
//func (s *Server) getCoreFiles(current string) []FileInfo {
//	var files []FileInfo
//
//	for _, id := range coreFilesSort {
//		if f, ok := coreFilesMap[id]; ok {
//			isCurrent := f.Path == current
//			isNavActive := isCurrent
//
//			// Special handling for daily/journal to paths that start with daily/ or journal/ as nav active
//			// when working with daily or journal files
//			if (id == "daily" || id == "journal") && strings.HasPrefix(current, id+"/") {
//				isNavActive = true
//			}
//
//			fileCopy := f
//			fileCopy.IsCurrent = isCurrent
//			fileCopy.IsNavActive = isNavActive
//			files = append(files, fileCopy)
//		}
//	}
//
//	return files
//}

//// getResourceFiles returns the cached list of resource files
//func (s *Server) getResourceFiles(current string) []FileInfo {
//	s.cacheMux.RLock()
//	defer s.cacheMux.RUnlock()
//
//	// Return a copy to avoid race conditions
//	filesCopy := make([]FileInfo, len(s.resourceCache))
//	copy(filesCopy, s.resourceCache)
//	return filesCopy
//}

//// buildDirectoryTree constructs a hierarchical tree of directories and files
//func (s *Server) buildDirectoryTree(files []FileInfo) *DirectoryNode {
//	root := &DirectoryNode{
//		Name:        "",
//		Files:       []FileInfo{},
//		Directories: make(map[string]*DirectoryNode),
//	}
//
//	for _, file := range files {
//		if file.Directory == "" {
//			// File is at the root of resources/
//			root.Files = append(root.Files, file)
//			continue
//		}
//
//		parts := strings.Split(file.Directory, string(filepath.Separator))
//		currentNode := root
//
//		for _, part := range parts {
//			if _, exists := currentNode.Directories[part]; !exists {
//				currentNode.Directories[part] = &DirectoryNode{
//					Name:        part,
//					Files:       []FileInfo{},
//					Directories: make(map[string]*DirectoryNode),
//				}
//			}
//			currentNode = currentNode.Directories[part]
//		}
//
//		currentNode.Files = append(currentNode.Files, file)
//	}
//
//	return root
//}

//// scanResourceFiles scans the resources directory for markdown files and returns their metadata
//func (s *Server) scanResourceFiles(current string) []FileInfo {
//	// Create the resources directory if it doesn't exist
//	if err := s.rootManager.MkdirAll(resourcesDir, 0755); err != nil {
//		log.Printf("Error creating resources directory: %v", err)
//		return []FileInfo{}
//	}
//
//	results, err := s.rootManager.Scan(resourcesDir, func(path string, d fs.DirEntry) bool {
//		return !d.IsDir() && strings.HasSuffix(d.Name(), ".md")
//	})
//
//	if err != nil {
//		log.Printf("Error scanning resources directory: %v", err)
//		return []FileInfo{}
//	}
//
//	var files []FileInfo
//	for _, result := range results {
//		// Create ID from relative path (replace separators)
//		//id := strings.ReplaceAll(result.Path, string(filepath.Separator), "_")
//		//id = strings.TrimSuffix(id, ".md")
//		id := s.createID(result.Path)
//
//		// Extract directory info
//		pathWithoutPrefix := strings.TrimPrefix(result.Path, resourcesDir+"/")
//		dir := filepath.Dir(pathWithoutPrefix)
//		if dir == "." {
//			dir = "" // Root of resources
//		}
//
//		// Calculate depth
//		depth := 0
//		if dir != "" {
//			depth = strings.Count(dir, string(filepath.Separator)) + 1
//		}
//
//		// Create display name
//		display := s.createDisplayName(result.Path)
//		displayBase := strings.TrimSuffix(filepath.Base(result.Name), ".md")
//		displayBase = strings.ReplaceAll(displayBase, "-", " ")
//		displayBase = strings.ReplaceAll(displayBase, "_", " ")
//		//goland:noinspection GoDeprecation
//		displayBase = strings.Title(displayBase)
//
//		files = append(files, FileInfo{
//			ID:          id,
//			Path:        result.Path,
//			Display:     display,
//			DisplayBase: displayBase,
//			IsCurrent:   result.Path == current,
//			Directory:   dir,
//			Depth:       depth,
//			IsResource:  true,
//		})
//	}
//
//	// Sort files alphabetically by display name for consistency
//	sort.Slice(files, func(i, j int) bool {
//		// Primary sort: Root files (empty directory) should come before any directory files
//		// This ensures all root-level files appear at the top, regardless of name
//
//		// Return true if i should come before j
//		if files[i].Directory == "" && files[j].Directory != "" {
//			return true
//		}
//
//		// Returning false here means j should come before i
//		if files[i].Directory != "" && files[j].Directory == "" {
//			return false
//		}
//
//		// Secondary sort: By directory name
//		if files[i].Directory != files[j].Directory {
//			return files[i].Directory < files[j].Directory
//		}
//
//		// Tertiary sort: By display name
//		return files[i].Display < files[j].Display
//	})
//
//	return files
//}

//// normalizeFileName creates a URL-safe, consistent filename/path
//func (s *Server) normalizeFileName(path string) string {
//	// Convert to lowercase for consistency
//	normalized := strings.ToLower(path)
//
//	// Always use forward slashes for URLs
//	normalized = strings.ReplaceAll(normalized, string(filepath.Separator), "/")
//
//	// Replace spaces and underscores with hyphens
//	normalized = strings.ReplaceAll(normalized, " ", "-")
//	normalized = strings.ReplaceAll(normalized, "_", "-")
//
//	// Remove or replace other problematic characters
//	// Keep only: letters, numbers, hyphens, periods, and forward slashes
//	var result strings.Builder
//	for _, char := range normalized {
//		switch {
//		case (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9'):
//			result.WriteRune(char)
//		case char == '-' || char == '.' || char == '/':
//			result.WriteRune(char)
//		default:
//			// Replace other characters with hyphens, but avoid consecutive hyphens
//			if result.Len() > 0 && result.String()[result.Len()-1] != '-' {
//				result.WriteRune('-')
//			}
//		}
//	}
//
//	// Clean up any trailing hyphens or multiple consecutive hyphens
//	cleaned := result.String()
//	cleaned = strings.Trim(cleaned, "-")
//
//	// Replace multiple consecutive hyphens with single hyphen
//	for strings.Contains(cleaned, "--") {
//		cleaned = strings.ReplaceAll(cleaned, "--", "-")
//	}
//
//	return cleaned
//}

//// createID generates a consistent URL-safe ID from a file path
//func (s *Server) createID(path string) string {
//	// Remove the .md extension and normalize
//	pathWithoutExt := strings.TrimSuffix(path, ".md")
//	normalized := s.normalizeFileName(pathWithoutExt)
//
//	//Remove resources/ prefix if present
//	//normalized = strings.TrimPrefix(normalized, "resources/")
//
//	return normalized
//}

//// createDisplayName generates a user-friendly display name from a file path
//func (s *Server) createDisplayName(relPath string) string {
//	// Remove the "resources/" prefix and ".md" suffix
//	pathWithoutPrefix := strings.TrimPrefix(relPath, resourcesDir+"/")
//	pathWithoutSuffix := strings.TrimSuffix(pathWithoutPrefix, ".md")
//
//	// Split into directory parts
//	parts := strings.Split(pathWithoutSuffix, string(filepath.Separator))
//
//	// Process each part: replace dashes/underscores with spaces and title case
//	for i, part := range parts {
//		part = strings.ReplaceAll(part, "-", " ")
//		part = strings.ReplaceAll(part, "_", " ")
//		//goland:noinspection GoDeprecation
//		parts[i] = strings.Title(part)
//	}
//
//	// Join with "/" to show hierarchy
//	return strings.Join(parts, "/")
//}

//// isValidFile checks if the requested file is a core file or a valid resource file
//func (s *Server) isValidFile(fileName string) bool {
//	// Check core files
//	for _, valid := range coreFilesMap {
//		if fileName == valid.Path {
//			return true
//		}
//	}
//
//	// Check for temporal files
//	if strings.HasPrefix(fileName, "daily/") || strings.HasPrefix(fileName, "journal/") {
//		return s.rootManager.FileExists(fileName)
//	}
//
//	if strings.HasPrefix(fileName, resourcesDir+"/") && strings.HasSuffix(fileName, ".md") {
//		return s.rootManager.FileExists(fileName)
//	}
//
//	return false
//}

//// getFileInfo retrieves FileInfo by ID, checking core and resource files
//func (s *Server) getFileInfo(id string) (FileInfo, error) {
//	if file, ok := coreFilesMap[id]; ok {
//		return file, nil
//	}
//
//	// Check for temporal files (e.g., daily/2025/09-september)
//	if strings.Contains(id, "/") {
//		now := time.Now()
//		parts := strings.Split(id, "/")
//		if len(parts) >= 2 && (parts[0] == "daily" || parts[0] == "journal") {
//			filePath := strings.Join(parts, "/") + ".md"
//			if s.rootManager.FileExists(filePath) {
//				monthPart := parts[2]
//				monthParts := strings.SplitN(monthPart, "-", 2)
//				if len(monthParts) == 2 {
//					displayName := fmt.Sprintf("%s %s", strings.Title(monthParts[1]), parts[1])
//
//					// Is this the current month?
//					isCurrent := parts[1] == now.Format("2006") && monthParts[0] == now.Format("01")
//
//					return FileInfo{
//						ID:          id,
//						Path:        filePath,
//						Display:     displayName,
//						DisplayBase: displayName,
//						IsCurrent:   isCurrent,
//						Directory:   parts[0] + "/" + parts[1],
//						Depth:       2,
//						IsResource:  false,
//						IsTemporal:  true,
//					}, nil
//				}
//			}
//		}
//	}
//
//	// Check resource files
//	resourceFiles := s.getResourceFiles(id)
//	for _, file := range resourceFiles {
//		if file.ID == id {
//			return file, nil
//		}
//	}
//
//	if id == "" {
//		return coreFilesMap["inbox"], nil
//	}
//
//	return FileInfo{}, fmt.Errorf("file with ID %s not found", id)
//}

//// getCurrentTemporalFile returns the FileInfo for the current month of a temporal file type
//func (s *Server) getCurrentTemporalFile(fileType string) (FileInfo, error) {
//	now := time.Now()
//	filePath, err := s.rootManager.ResolveMonthlyFile(now, fileType)
//	if err != nil {
//		return FileInfo{}, err
//	}
//
//	id := s.createID(filePath)
//	displayName := fmt.Sprintf("%s %d", now.Format("January"), now.Year())
//
//	return FileInfo{
//		ID:          id,
//		Path:        filePath,
//		Display:     displayName,
//		DisplayBase: displayName,
//		IsCurrent:   false,
//		IsTemporal:  true,
//	}, nil
//}

//// getTemporalFiles lists all existing temporal files of a given type (daily or journal)
//func (s *Server) getTemporalFiles(fileType string) ([]string, map[string][]FileInfo, error) {
//	var yearKeys []string
//	files := make(map[string][]FileInfo)
//
//	// Check if the directory exists
//	yearEntries, err := s.rootManager.ReadDir(fileType)
//	if err != nil {
//		return yearKeys, files, nil // Return empty list if directory doesn't exist
//	}
//
//	for _, yearEntry := range yearEntries {
//		if !yearEntry.IsDir() {
//			continue
//		}
//
//		yearPath := filepath.Join(fileType, yearEntry.Name())
//		monthEntries, err := s.rootManager.ReadDir(yearPath)
//		if err != nil {
//			continue // Skip this year if there's an error
//		}
//
//		// Create the year entry if it doesn't exist
//		if _, exists := files[yearEntry.Name()]; !exists {
//			files[yearEntry.Name()] = []FileInfo{}
//		}
//
//		for _, monthEntry := range monthEntries {
//			if !monthEntry.IsDir() && strings.HasSuffix(monthEntry.Name(), ".md") {
//				monthName := strings.TrimSuffix(monthEntry.Name(), ".md")
//				filePath := filepath.Join(yearPath, monthEntry.Name())
//				id := fmt.Sprintf("%s/%s/%s", fileType, yearEntry.Name(), monthName)
//
//				parts := strings.SplitN(monthName, "-", 2)
//				displayName := monthName // Fallback to raw month name
//				monthNumber := parts[0]
//				monthDisplay := monthName
//				if len(parts) == 2 {
//					displayName = fmt.Sprintf("%s %s", strings.Title(parts[1]), yearEntry.Name())
//					monthDisplay = strings.Title(parts[1])
//				}
//
//				files[yearEntry.Name()] = append(files[yearEntry.Name()], FileInfo{
//					ID:          id,
//					Path:        filePath,
//					Display:     displayName,
//					DisplayBase: displayName,
//					Directory:   fileType + "/" + yearEntry.Name(),
//					Year:        yearEntry.Name(),
//					Month:       monthNumber,
//					MonthName:   monthDisplay,
//				})
//			}
//		}
//
//		// Sort months within the year
//		sort.Slice(files[yearEntry.Name()], func(i, j int) bool {
//			return files[yearEntry.Name()][i].Month > files[yearEntry.Name()][j].Month // Reverse chronological order
//		})
//	}
//
//	yearKeys = slices.Sorted(maps.Keys(files))
//	slices.Reverse(yearKeys)
//
//	return yearKeys, files, nil
//}
