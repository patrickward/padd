package files

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type Task struct {
	ID        int
	Label     string
	IsChecked bool
	LineIndex int
	Prefix    string // e.g., "- " or "* "
	State     string // " " or "x" or "X"
	Suffix    string // The rest of the line
}

//goland:noinspection RegExpRedundantEscape
var taskListPattern = regexp.MustCompile(`^(\s*[-*]\s+)\[([ xX])\](.*)$`)

func (d *Document) GetTask(taskID int) (*Task, error) {
	return d.findTaskByID(taskID)
}

func (d *Document) ToggleTask(taskID int) (*Task, error) {
	task, err := d.findTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(d.content, "\n")

	var newState string
	var newSuffix string

	if strings.TrimSpace(task.State) == "" {
		newState = "x"
		newSuffix = strings.TrimSpace(task.Suffix) + fmt.Sprintf(" @done(%s)", time.Now().Format("2006-01-02"))
	} else {
		newState = " "
		newSuffix = regexp.MustCompile(`\s*@done\(\d{4}-\d{2}-\d{2}\)`).ReplaceAllString(task.Suffix, "")
	}

	lines[task.LineIndex] = fmt.Sprintf("%s[%s] %s", task.Prefix, newState, newSuffix)

	updatedContent := strings.Join(lines, "\n")
	if err := d.Save(updatedContent); err != nil {
		return nil, fmt.Errorf("failed to save: %w", err)
	}

	// Return the updated task
	task.Label = strings.TrimSpace(newSuffix)
	task.IsChecked = newState == "x"
	task.State = newState
	task.Suffix = newSuffix

	return task, nil
}

func (d *Document) UpdateTaskLabel(taskID int, newLabel string) (*Task, error) {
	task, err := d.findTaskByID(taskID)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(d.content, "\n")

	// If task is checked and doesn't have @done tag, add it
	if task.IsChecked {
		doneTags := regexp.MustCompile(`\s*@done\(\d{4}-\d{2}-\d{2}\)`).FindString(newLabel)
		if doneTags == "" {
			newLabel += fmt.Sprintf(" @done(%s)", time.Now().Format("2006-01-02"))
		}
	}

	newSuffix := " " + strings.TrimSpace(newLabel)
	lines[task.LineIndex] = fmt.Sprintf("%s[%s]%s", task.Prefix, task.State, newSuffix)

	updatedContent := strings.Join(lines, "\n")
	if err := d.Save(updatedContent); err != nil {
		return nil, fmt.Errorf("failed to save: %w", err)
	}

	task.Label = strings.TrimSpace(newSuffix)
	task.Suffix = newSuffix

	return task, nil
}

func (d *Document) DeleteTask(taskID int) error {
	task, err := d.findTaskByID(taskID)
	if err != nil {
		return err
	}

	lines := strings.Split(d.content, "\n")

	// Remove the line at task.LineIndex
	lines = append(lines[:task.LineIndex], lines[task.LineIndex+1:]...)
	updatedContent := strings.Join(lines, "\n")

	return d.Save(updatedContent)
}

func (d *Document) ArchiveCompletedTasks() ([]string, error) {
	if err := d.load(); err != nil {
		return nil, err
	}

	var remainingLines []string
	var completedTasks []string
	lines := strings.Split(d.content, "\n")

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

	if len(completedTasks) > 0 {
		updatedContent := strings.Join(remainingLines, "\n")
		if err := d.Save(updatedContent); err != nil {
			return nil, fmt.Errorf("failed to save: %w", err)
		}
	}

	return completedTasks, nil
}

func (d *Document) findTaskByID(taskID int) (*Task, error) {
	tasks, err := d.getAllTasks()
	if err != nil {
		return nil, err
	}

	if taskID < 1 || taskID > len(tasks) {
		return nil, fmt.Errorf("task ID %d not found (document has %d tasks)", taskID, len(tasks))
	}

	return &tasks[taskID-1], nil
}

// extractAllTasks extracts all tasks from the given lines.
func (d *Document) extractAllTasks(lines []string) []Task {
	var tasks []Task
	taskCount := 0

	for i, line := range lines {
		if matches := taskListPattern.FindStringSubmatch(line); matches != nil {
			taskCount++
			prefix := matches[1]
			state := matches[2]
			suffix := matches[3]
			label := strings.TrimSpace(suffix)
			isChecked := state == "x" || state == "X"
			tasks = append(tasks, Task{
				ID:        taskCount,
				Label:     label,
				IsChecked: isChecked,
				LineIndex: i,
				Prefix:    prefix,
				State:     state,
				Suffix:    suffix,
			})
		}
	}

	return tasks
}

func (d *Document) getAllTasks() ([]Task, error) {
	if err := d.load(); err != nil {
		return nil, err
	}

	d.taskMu.RLock()
	if d.taskCacheValid && d.taskCache != nil {
		d.taskMu.RUnlock()
		return d.taskCache, nil
	}
	d.taskMu.RUnlock()

	// Rebuild the cache
	d.taskMu.Lock()
	defer d.taskMu.Unlock()

	// Double-check cache validity (another goroutine may have already updated it)
	if d.taskCacheValid && d.taskCache != nil {
		return d.taskCache, nil
	}

	lines := strings.Split(d.content, "\n")
	d.taskCache = d.extractAllTasks(lines)
	d.taskCacheValid = true

	return d.taskCache, nil
}

func (d *Document) invalidateTaskCache() {
	d.taskMu.Lock()
	defer d.taskMu.Unlock()

	d.taskCacheValid = false
	d.taskCache = nil
}
