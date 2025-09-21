package padd_test

import (
	"strings"
	"testing"
	"time"

	"github.com/patrickward/padd"
	"github.com/patrickward/padd/internal/assert"
)

func TestDocument_AddEntry_InsertByTimestamp_NewerDate(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	// Create a temporal document (daily) with existing content that has older dates
	baseTime := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	doc, err := fr.GetOrCreateTemporalDocument("daily", baseTime)
	assert.Nil(t, err)

	initialContent := `# Daily September 2025

## Monday, September 15, 2025

### 10:00:00 AM

Older entry 1

## Sunday, September 14, 2025

### 09:00:00 AM

Oldest entry`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	// Add entry with newer timestamp (should go at the top)
	newerTime := time.Date(2025, 9, 16, 14, 30, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: newerTime,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("Newer entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should contain the new date header at the top
	assert.True(t, strings.Contains(content, "## Tuesday, September 16, 2025"))
	assert.True(t, strings.Contains(content, "### 02:30:00 PM"))
	assert.True(t, strings.Contains(content, "Newer entry"))

	// Verify order: newest first
	lines := padd.SplitLines(content)
	var headerIndices []int
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headerIndices = append(headerIndices, i)
		}
	}

	assert.Equal(t, len(headerIndices), 3)
	// Check the order of headers
	assert.True(t, strings.Contains(lines[headerIndices[0]], "September 16, 2025")) // Newest
	assert.True(t, strings.Contains(lines[headerIndices[1]], "September 15, 2025")) // Middle
	assert.True(t, strings.Contains(lines[headerIndices[2]], "September 14, 2025")) // Oldest
}

func TestDocument_AddEntry_InsertByTimestamp_MiddleDate(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	baseTime := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	doc, err := fr.GetOrCreateTemporalDocument("daily", baseTime)
	assert.Nil(t, err)

	initialContent := `# Daily September 2025

## Wednesday, September 17, 2025

### 10:00:00 AM

Newer entry

## Monday, September 15, 2025

### 09:00:00 AM

Older entry`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	// Add entry with middle timestamp (should go between existing dates)
	middleTime := time.Date(2025, 9, 16, 12, 0, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: middleTime,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("Middle entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Verify the middle date header is inserted
	assert.True(t, strings.Contains(content, "## Tuesday, September 16, 2025"))
	assert.True(t, strings.Contains(content, "Middle entry"))

	// Verify chronological order
	lines := padd.SplitLines(content)
	var headerIndices []int
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headerIndices = append(headerIndices, i)
		}
	}

	assert.Equal(t, len(headerIndices), 3)
	assert.True(t, strings.Contains(lines[headerIndices[0]], "September 17, 2025")) // Newest
	assert.True(t, strings.Contains(lines[headerIndices[1]], "September 16, 2025")) // Middle
	assert.True(t, strings.Contains(lines[headerIndices[2]], "September 15, 2025")) // Oldest
}

func TestDocument_AddEntry_InsertByTimestamp_OlderDate(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	baseTime := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	doc, err := fr.GetOrCreateTemporalDocument("daily", baseTime)
	assert.Nil(t, err)

	initialContent := `# Daily September 2025

## Tuesday, September 16, 2025

### 10:00:00 AM

Newer entry

## Monday, September 15, 2025

### 09:00:00 AM

Middle entry`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	// Add entry with older timestamp (should go at the bottom)
	olderTime := time.Date(2025, 9, 14, 8, 0, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: olderTime,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("Older entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Verify the older date header is added at the bottom
	assert.True(t, strings.Contains(content, "## Sunday, September 14, 2025"))
	assert.True(t, strings.Contains(content, "Older entry"))

	// Verify chronological order
	lines := padd.SplitLines(content)
	var headerIndices []int
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headerIndices = append(headerIndices, i)
		}
	}

	assert.Equal(t, len(headerIndices), 3)
	assert.True(t, strings.Contains(lines[headerIndices[0]], "September 16, 2025")) // Newest
	assert.True(t, strings.Contains(lines[headerIndices[1]], "September 15, 2025")) // Middle
	assert.True(t, strings.Contains(lines[headerIndices[2]], "September 14, 2025")) // Oldest
}

func TestDocument_AddEntry_InsertByTimestamp_ExistingDate(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	baseTime := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	doc, err := fr.GetOrCreateTemporalDocument("daily", baseTime)
	assert.Nil(t, err)

	initialContent := `# Daily September 2025

## Monday, September 15, 2025

### 10:00:00 AM

First entry

## Sunday, September 14, 2025

### 09:00:00 AM

Second entry`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	// Add entry to existing date (should be added under existing header)
	existingTime := time.Date(2025, 9, 15, 15, 30, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: existingTime,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("Another entry for same day", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should not create a new date header
	headerCount := strings.Count(content, "## Monday, September 15, 2025")
	assert.Equal(t, headerCount, 1)

	// Should contain both entries under the same date
	assert.True(t, strings.Contains(content, "First entry"))
	assert.True(t, strings.Contains(content, "Another entry for same day"))
	assert.True(t, strings.Contains(content, "### 03:30:00 PM"))
}

func TestDocument_AddEntry_InsertByTimestamp_EmptyFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	doc, err := fr.GetOrCreateResourceDocument("empty-doc")
	assert.Nil(t, err)

	// Add entry to empty file
	timestamp := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: timestamp,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("First entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Assert equal, but ignore all whitespace
	content = strings.ReplaceAll(content, "\n", "")
	content = strings.ReplaceAll(content, " ", "")
	assert.Equal(t, content, "#empty-doc.md##Monday,September15,2025###10:00:00AMFirstentry")
}

func TestDocument_AddEntry_InsertByTimestamp_NoMainHeader(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	baseTime := time.Date(2025, 9, 15, 10, 0, 0, 0, time.UTC)
	doc, err := fr.GetOrCreateTemporalDocument("daily", baseTime)
	assert.Nil(t, err)

	// File with content but no main header
	initialContent := `Some initial content without main header

## Monday, September 15, 2025

### 10:00:00 AM

Existing entry`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	// Add entry with newer timestamp
	newerTime := time.Date(2025, 9, 16, 14, 30, 0, 0, time.UTC)
	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertByTimestamp,
		EntryTimestamp: newerTime,
		EntryFormatter: padd.TimestampEntryFormatter,
	}

	err = doc.AddEntry("New entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should add the new date at the top since no main header found
	assert.True(t, strings.Contains(content, "## Tuesday, September 16, 2025"))
	assert.True(t, strings.Contains(content, "New entry"))

	// Verify it's at the beginning
	lines := padd.SplitLines(content)
	assert.True(t, strings.Contains(lines[2], "## Tuesday, September 16, 2025"))
}

func TestDocument_AddEntry_PrependToFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	doc, err := fr.GetOrCreateResourceDocument("test-doc")
	assert.Nil(t, err)

	initialContent := `# Test Document

Existing content`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	config := padd.EntryInsertionConfig{
		Strategy:       padd.PrependToFile,
		EntryFormatter: padd.NoteEntryFormatter,
	}

	err = doc.AddEntry("Prepended entry", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should be at the very beginning
	assert.True(t, strings.HasPrefix(content, "Prepended entry\n"))
	assert.True(t, strings.Contains(content, "# Test Document"))
}

func TestDocument_AddEntry_AppendToFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	doc, err := fr.GetOrCreateResourceDocument("test-doc")
	assert.Nil(t, err)

	initialContent := `# Test Document

Existing content`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	config := padd.EntryInsertionConfig{
		Strategy:       padd.AppendToFile,
		EntryFormatter: padd.TaskEntryFormatter,
	}

	err = doc.AddEntry("Appended task", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should be at the end
	assert.True(t, strings.HasSuffix(strings.TrimSpace(content), "- [ ] Appended task"))
	assert.True(t, strings.Contains(strings.TrimSpace(content), "# Test Document"))
}

func TestDocument_AddEntry_InsertInSection_ExistingSection(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	doc, err := fr.GetOrCreateResourceDocument("test-doc")
	assert.Nil(t, err)

	initialContent := `# Test Document

## Tasks

- [ ] Existing task

## Notes

Some existing note`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertInSection,
		EntryFormatter: padd.TaskEntryFormatter,
		SectionConfig: &padd.SectionInsertionConfig{
			SectionHeader:  "## Tasks",
			InsertAtTop:    true,
			BlankLineAfter: false,
		},
	}

	err = doc.AddEntry("New task", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should be inserted in the Tasks section
	assert.True(t, strings.Contains(content, "- [ ] New task"))
	assert.True(t, strings.Contains(content, "- [ ] Existing task"))

	// Verify order (new task should be first)
	lines := padd.SplitLines(content)
	var taskLines []int
	for i, line := range lines {
		if strings.HasPrefix(line, "- [ ]") {
			taskLines = append(taskLines, i)
		}
	}

	assert.Equal(t, len(taskLines), 2)
	assert.True(t, strings.Contains(lines[taskLines[0]], "New task"))
	assert.True(t, strings.Contains(lines[taskLines[1]], "Existing task"))
}

func TestDocument_AddEntry_InsertInSection_NewSection(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches to ensure we've initialized
	fr.ReloadCaches()

	doc, err := fr.GetOrCreateResourceDocument("test-doc")
	assert.Nil(t, err)

	initialContent := `# Test Document

Some existing content`

	err = doc.Save(initialContent)
	assert.Nil(t, err)

	config := padd.EntryInsertionConfig{
		Strategy:       padd.InsertInSection,
		EntryFormatter: padd.TaskEntryFormatter,
		SectionConfig: &padd.SectionInsertionConfig{
			SectionHeader:  "## New Tasks",
			InsertAtTop:    true,
			BlankLineAfter: true,
		},
	}

	err = doc.AddEntry("First task in new section", config)
	assert.Nil(t, err)

	content, err := doc.Content()
	assert.Nil(t, err)

	// Should create the new section
	assert.True(t, strings.Contains(content, "## New Tasks"))
	assert.True(t, strings.Contains(content, "- [ ] First task in new section"))

	// Verify the section was created at the top (after main header)
	lines := padd.SplitLines(content)
	found := false
	for i, line := range lines {
		if strings.Contains(line, "# Test Document") && i+2 < len(lines) {
			// Skip potential blank lines
			nextContentIdx := i + 1
			for nextContentIdx < len(lines) && strings.TrimSpace(lines[nextContentIdx]) == "" {
				nextContentIdx++
			}
			if nextContentIdx < len(lines) && strings.Contains(lines[nextContentIdx], "## New Tasks") {
				found = true
				break
			}
		}
	}
	assert.True(t, found)
}
