package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/patrickward/padd"
)

var reloadPageHeaderTrigger = map[string]string{
	"HX-Trigger": "padd:reload-header",
}

// handleTaskToggle toggles the completion state of a task item in a Markdown file.
func (s *Server) handleTaskToggle(w http.ResponseWriter, r *http.Request) {
	doc, checkboxID, done := s.taskDocumentFromRequest(w, r)
	if done {
		return
	}

	task, err := doc.ToggleTask(checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = s.executeSnippetWithHeaders(w, "task_show", map[string]any{
		"ID":              task.ID,
		"Value":           task.Label,
		"IsChecked":       task.IsChecked,
		"IncludeCheckbox": false,
	}, reloadPageHeaderTrigger); err != nil {
		s.showServerError(w, r, err)
	}
}

// handleTaskShow renders a task item for display.
func (s *Server) handleTaskShow(w http.ResponseWriter, r *http.Request) {
	doc, checkboxID, done := s.taskDocumentFromRequest(w, r)
	if done {
		return
	}

	task, err := doc.GetTask(checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.executeSnippet(w, "task_show", map[string]any{
		"ID":              task.ID,
		"Value":           task.Label,
		"IsChecked":       task.IsChecked,
		"IncludeCheckbox": true,
	}); err != nil {
		s.showServerError(w, r, err)
	}
}

// handleTaskEdit renders a form to edit a task item.
func (s *Server) handleTaskEdit(w http.ResponseWriter, r *http.Request) {
	doc, checkboxID, done := s.taskDocumentFromRequest(w, r)
	if done {
		return
	}

	task, err := doc.GetTask(checkboxID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.executeSnippetWithHeaders(w, "task_edit", map[string]any{
		"ID":    task.ID,
		"Value": task.Label,
	}, reloadPageHeaderTrigger); err != nil {
		s.showServerError(w, r, err)
	}
}

// handleTaskUpdate updates the label of a task item in a Markdown file.
func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	doc, checkboxID, done := s.taskDocumentFromRequest(w, r)
	if done {
		return
	}

	newLabel := r.FormValue("label")
	if newLabel == "" {
		http.Error(w, "Missing label parameter", http.StatusBadRequest)
		return
	}

	task, err := doc.UpdateTaskLabel(checkboxID, newLabel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.executeSnippetWithHeaders(w, "task_show", map[string]any{
		"ID":              task.ID,
		"Value":           task.Label,
		"IsChecked":       task.IsChecked,
		"IncludeCheckbox": true,
	}, reloadPageHeaderTrigger); err != nil {
		s.showServerError(w, r, err)
	}
}

// handleTaskDelete removes a task item from a Markdown file.
func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	doc, checkboxID, done := s.taskDocumentFromRequest(w, r)
	if done {
		return
	}

	if err := doc.DeleteTask(checkboxID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add the HX-Refresh header to refresh the task list. This ensures sequential IDs are updated.
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent)
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

	doc, err := s.fileRepo.GetDocument(fileID)
	if err != nil {
		s.flashManager.SetError(w, "Invalid file.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// Skip temporal files
	if s.fileRepo.FileIsTemporal(doc.Info.Path) {
		s.flashManager.SetError(w, "Cannot archive tasks from temporal files.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	completedTasks, err := doc.ArchiveCompletedTasks()
	if err != nil {
		s.flashManager.SetError(w, "Failed to archive tasks: "+err.Error())
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	if len(completedTasks) == 0 {
		s.flashManager.SetSuccess(w, "No completed tasks to archive.")
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// Add archived tasks to today's daily file
	dailyDoc, err := s.fileRepo.GetOrCreateTemporalDocument("daily", time.Now())
	if err != nil {
		s.flashManager.SetError(w, "Failed to get daily document: "+err.Error())
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	archivedContent := "**Archived completed tasks (from " + doc.Info.Path + "):**\n\n"
	archivedContent += strings.Join(completedTasks, "\n")

	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	if err := dailyDoc.AddEntry(archivedContent, config); err != nil {
		s.flashManager.SetError(w, "Failed to add archived tasks to daily file: "+err.Error())
		w.Header().Set("HX-Redirect", r.Header.Get("Referer"))
		w.WriteHeader(http.StatusSeeOther)
		return
	}

	// Set a flash message indicating how many tasks were archived
	s.flashManager.SetSuccess(w, fmt.Sprintf("Archived %d completed task(s).", len(completedTasks)))

	// Add the HX-Refresh header to refresh the task list.
	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusNoContent) // 204 No Content
}

// taskDocumentFromRequest returns the document and checkbox ID from the request.
func (s *Server) taskDocumentFromRequest(w http.ResponseWriter, r *http.Request) (*padd.Document, int, bool) {
	fileID := r.Header.Get("X-PADD-File-ID")
	if fileID == "" {
		http.Error(w, "Missing file ID", http.StatusBadRequest)
		return nil, 0, true
	}

	doc, err := s.fileRepo.GetDocument(fileID)
	if err != nil {
		http.Error(w, "Invalid file", http.StatusBadRequest)
	}

	checkboxIDStr := r.PathValue("id")
	if checkboxIDStr == "" {
		http.Error(w, "Missing checkbox_id parameter", http.StatusBadRequest)
		return nil, 0, true
	}

	checkboxID := 0
	if _, err := fmt.Sscanf(checkboxIDStr, "%d", &checkboxID); err != nil || checkboxID <= 0 {
		http.Error(w, "Invalid checkbox_id parameter", http.StatusBadRequest)
		return nil, 0, true
	}

	return doc, checkboxID, false
}
