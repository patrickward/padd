package padd

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

var taskListPattern = regexp.MustCompile(`^(\s*[-*]\s+)\[([ xX])\](.*)$`)

func (d *Document) findTaskByID(taskID int) (*Task, error) {
	if err := d.load(); err != nil {
		return nil, err
	}

	taskCount := 0
	lines := strings.Split(d.content, "\n")

	for i, line := range lines {
		if matches := taskListPattern.FindStringSubmatch(line); matches != nil {
			taskCount++
			if taskCount == taskID {
				prefix := matches[1]
				state := matches[2]
				suffix := matches[3]
				label := strings.TrimSpace(suffix)
				isChecked := state == "x" || state == "X"

				return &Task{
					ID:        taskID,
					Label:     label,
					IsChecked: isChecked,
					LineIndex: i,
					Prefix:    prefix,
					State:     state,
					Suffix:    suffix,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("task ID %d not found", taskID)
}

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
