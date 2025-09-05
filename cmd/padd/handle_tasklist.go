package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var reloadPageHeaderTrigger = map[string]string{
	"HX-Trigger": "padd:reload-header",
}

// handleTaskToggle toggles the completion state of a task item in a Markdown file.
func (s *Server) handleTaskToggle(w http.ResponseWriter, r *http.Request) {
	file, checkboxID, content, done := s.extractTaskInfoFromRequest(w, r)
	if done {
		return
	}

	updatedContent, isChecked, label, err := s.toggleTask(string(content), checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.dirManager.WriteString(file.Path, updatedContent); err != nil {
		s.showServerError(w, r, err)
		return
	}

	err = s.executeSnippetWithHeaders(w, "task_show", map[string]any{
		"ID":              checkboxID,
		"Value":           label,
		"IsChecked":       isChecked,
		"IncludeCheckbox": false,
	}, reloadPageHeaderTrigger)

	if err != nil {
		s.showServerError(w, r, err)
		return
	}
}

// handleTaskShow renders a task item for display.
func (s *Server) handleTaskShow(w http.ResponseWriter, r *http.Request) {
	_, checkboxID, content, done := s.extractTaskInfoFromRequest(w, r)
	if done {
		return
	}

	_, matches, err := s.findTaskBySequentialID(string(content), checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	currentState := matches[2] // " " or "x" or "X"
	suffix := matches[3]       // The rest of the line
	isChecked := strings.TrimSpace(currentState) != ""

	err = s.executeSnippet(w, "task_show", map[string]any{
		"ID":              checkboxID,
		"Value":           suffix,
		"IsChecked":       isChecked,
		"IncludeCheckbox": true,
	})
}

// handleTaskEdit renders a form to edit a task item.
func (s *Server) handleTaskEdit(w http.ResponseWriter, r *http.Request) {
	_, checkboxID, content, done := s.extractTaskInfoFromRequest(w, r)
	if done {
		return
	}

	// Extract the current task label
	currentLabel, err := s.extractTaskLabel(string(content), checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.executeSnippetWithHeaders(w, "task_edit", map[string]any{
		"ID":    checkboxID,
		"Value": currentLabel,
	}, reloadPageHeaderTrigger)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}
}

// handleTaskUpdate updates the label of a task item in a Markdown file.
func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	file, checkboxID, content, done := s.extractTaskInfoFromRequest(w, r)
	if done {
		return
	}

	newLabel := r.FormValue("label")
	if newLabel == "" {
		http.Error(w, "Missing label parameter", http.StatusBadRequest)
		return
	}

	updatedContent, updatedLabel, isChecked, err := s.updateTaskLabel(string(content), checkboxID, newLabel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.dirManager.WriteString(file.Path, updatedContent); err != nil {
		s.showServerError(w, r, err)
		return
	}

	err = s.executeSnippetWithHeaders(w, "task_show", map[string]any{
		"ID":              checkboxID,
		"Value":           updatedLabel,
		"IsChecked":       isChecked,
		"IncludeCheckbox": true,
	}, reloadPageHeaderTrigger)
	if err != nil {
		s.showServerError(w, r, err)
		return
	}
}

// handleTaskDelete removes a task item from a Markdown file.
func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	file, checkboxID, content, done := s.extractTaskInfoFromRequest(w, r)
	if done {
		return
	}

	lines := strings.Split(string(content), "\n")
	lineIndex, _, err := s.findTaskBySequentialID(string(content), checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Remove the line at lineIndex
	lines = append(lines[:lineIndex], lines[lineIndex+1:]...)
	updatedContent := strings.Join(lines, "\n")

	if err := s.dirManager.WriteString(file.Path, updatedContent); err != nil {
		s.showServerError(w, r, err)
		return
	}

	// Add the HX-Refresh header to refresh the task list. This ensures sequential IDs are updated.
	// Otherwise, when subsequent tasks are toggled/edited, the IDs would be incorrect.
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// extractTaskInfoFromRequest extracts the file info, checkbox ID, and file content from the request.
func (s *Server) extractTaskInfoFromRequest(w http.ResponseWriter, r *http.Request) (FileInfo, int, []byte, bool) {
	fileID := r.Header.Get("X-PADD-File-ID")
	if fileID == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return FileInfo{}, 0, nil, true
	}

	file, err := s.getFileInfo(fileID)
	if err != nil || !s.isValidFile(file.Path) {
		http.Error(w, "Invalid file", http.StatusBadRequest)
		return FileInfo{}, 0, nil, true
	}

	//checkboxIDStr := r.FormValue("checkbox_id")
	checkboxIDStr := r.PathValue("id")

	if checkboxIDStr == "" {
		http.Error(w, "Missing checkbox_id parameter", http.StatusBadRequest)
		return FileInfo{}, 0, nil, true
	}

	checkboxID := 0
	if _, err := fmt.Sscanf(checkboxIDStr, "%d", &checkboxID); err != nil || checkboxID <= 0 {
		http.Error(w, "Invalid checkbox_id parameter", http.StatusBadRequest)
		return FileInfo{}, 0, nil, true
	}

	content, err := s.dirManager.ReadFile(file.Path)
	if err != nil {
		s.showServerError(w, r, err)
		return FileInfo{}, 0, nil, true
	}

	return file, checkboxID, content, false
}

// findTaskBySequentialID finds the line index and regex matches of a task item by its sequential checkbox ID.
func (s *Server) findTaskBySequentialID(content string, checkboxID int) (lineIndex int, matches []string, err error) {
	lines := strings.Split(content, "\n")
	checkboxCount := 0

	// Regex pattern to match task list items
	taskListPattern := regexp.MustCompile(`^(\s*[-*]\s+)\[([ xX])\](.*)$`)

	for i, line := range lines {
		matches := taskListPattern.FindStringSubmatch(line)
		if matches != nil {
			checkboxCount++
			if checkboxCount == checkboxID {
				return i, matches, nil
			}
		}
	}

	return -1, nil, fmt.Errorf("checkbox ID %d not found", checkboxID)
}

// extractTaskLabel extracts the label of a task item by its sequential checkbox ID.
func (s *Server) extractTaskLabel(content string, checkboxID int) (string, error) {
	_, matches, err := s.findTaskBySequentialID(content, checkboxID)
	if err != nil {
		return "", err
	}

	// Extract the label from the  matches
	label := strings.TrimSpace(matches[3])

	return label, nil
}

// updateTaskLabel updates the label of a task item by its sequential checkbox ID.
func (s *Server) updateTaskLabel(content string, checkboxID int, newLabel string) (updatedContent string, label string, isChecked bool, err error) {
	lines := strings.Split(content, "\n")
	lineIndex, matches, err := s.findTaskBySequentialID(content, checkboxID)
	if err != nil {
		return content, "", false, err
	}

	prefix := matches[1]       // e.g., "- " or "* "
	currentState := matches[2] // " " or "x" or "X"

	// Reconstruct the line with the new label
	isChecked = strings.TrimSpace(currentState) != ""

	// If there is no @done tag and the task is marked done, add the current date
	var doneTags string
	if isChecked {
		doneTags = regexp.MustCompile(`\s*@done\(\d{4}-\d{2}-\d{2}\)`).FindString(newLabel)
		if doneTags == "" {
			newLabel += fmt.Sprintf(" @done(%s)", time.Now().Format("2006-01-02"))
		}
	}

	newSuffix := " " + strings.TrimSpace(newLabel)
	lines[lineIndex] = fmt.Sprintf("%s[%s]%s", prefix, currentState, newSuffix)
	return strings.Join(lines, "\n"), newSuffix, isChecked, nil
}

// toggleTask toggles the completion state of a task item by its sequential checkbox ID.
func (s *Server) toggleTask(content string, checkboxID int) (updatedContent string, isChecked bool, label string, err error) {
	lines := strings.Split(content, "\n")
	lineIndex, matches, err := s.findTaskBySequentialID(content, checkboxID)
	if err != nil {
		return content, false, "", err
	}

	prefix := matches[1]       // e.g., "- " or "* "
	currentState := matches[2] // " " or "x" or "X"
	suffix := matches[3]       // The rest of the line

	var newState string
	if strings.TrimSpace(currentState) == "" {
		newState = "x"
		// Add @done(YYYY-MM-DD) tag
		suffix = strings.TrimSpace(suffix) + fmt.Sprintf(" @done(%s)", time.Now().Format("2006-01-02"))
	} else {
		newState = " "
		// Remove any existing @done(...) tags
		suffix = regexp.MustCompile(`\s*@done\(\d{4}-\d{2}-\d{2}\)`).ReplaceAllString(suffix, "")
	}

	// Reconstruct the line with the toggled state
	isChecked = newState == "x"
	lines[lineIndex] = fmt.Sprintf("%s[%s]%s", prefix, newState, suffix)
	return strings.Join(lines, "\n"), isChecked, suffix, nil
}

// handleArchiveDoneTasks archives all completed tasks from a specified file to their respective daily files.
func (s *Server) handleArchiveDoneTasks(w http.ResponseWriter, r *http.Request) {
	fileID := r.PathValue("id")
	if fileID == "" {
		s.flashManager.SetError(w, "File ID is required.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	file, err := s.getFileInfo(fileID)
	if err != nil || !s.isValidFile(file.Path) {
		s.flashManager.SetError(w, "Invalid file.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// Skip temporal files
	if s.isTemporalFile(file.Path) {
		s.flashManager.SetError(w, "Cannot archive tasks from temporal files.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	contentBytes, err := s.dirManager.ReadFile(file.Path)
	if err != nil {
		s.flashManager.SetError(w, "Failed to read file.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	archivedCount, updatedContent, err := s.archiveCompletedTasks(string(contentBytes), file.Path)
	if err != nil {
		s.flashManager.SetError(w, "Failed to archive tasks: "+err.Error())
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	if archivedCount == 0 {
		s.flashManager.SetSuccess(w, "No completed tasks to archive.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	if err := s.dirManager.WriteString(file.Path, updatedContent); err != nil {
		s.flashManager.SetError(w, "Failed to update file.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// Set a flash message indicating how many tasks were archived
	s.flashManager.SetSuccess(w, fmt.Sprintf("Archived %d completed task(s).", archivedCount))

	// Add the HX-Refresh header to refresh the task list.
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// archiveCompletedTasks processes the content of a Markdown file, archives completed tasks to their respective daily files,
func (s *Server) archiveCompletedTasks(content string, sourceFilePath string) (int, string, error) {
	lines := strings.Split(content, "\n")
	var remainingLines []string
	var completedTasks []string

	taskListPattern := regexp.MustCompile(`^(\s*[-*]\s+)\[([ xX])\](.*)$`)

	for _, line := range lines {
		if matches := taskListPattern.FindStringSubmatch(line); matches != nil && strings.TrimSpace(matches[2]) != "" {
			// This is a completed task
			taskText := strings.TrimSpace(matches[3])
			archiveEntry := fmt.Sprintf("- &#x2713; %s", taskText)
			completedTasks = append(completedTasks, archiveEntry)
		} else {
			remainingLines = append(remainingLines, line)
		}
	}

	// Archive all tasks to today's daily file. Good enough for now.
	// This is a simple compromise for now, maybe later we can move them to the date they were marked done.
	// Keep in mind that may require additional work on the entries, such as ensuring dates are properly sorted.
	now := time.Now()
	if len(completedTasks) > 0 {
		completedContent := "**Archived completed tasks (from " + sourceFilePath + "):**\n\n"
		completedContent += strings.Join(completedTasks, "\n") + "\n"
		if err := s.addTemporalEntry(completedContent, "daily", now); err != nil {
			return 0, content, fmt.Errorf("failed to archive tasks: %w", err)
		}
	}

	return len(completedTasks), strings.Join(remainingLines, "\n"), nil
}
