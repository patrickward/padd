package files_test

import (
	"testing"

	"github.com/patrickward/padd/internal/assert"
	"github.com/patrickward/padd/internal/files"
)

func TestCSVDocument_BasicOperations(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Create a test CSV file
	csvContent := "name,age,city\nJohn,25,NYC\nJane,30,LA\nBob,35,Chicago\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	// Create a Document and wrap it in CSVDocument
	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Test getting records
	records, err := csvDoc.GetRecords()
	assert.Nil(t, err)
	assert.Equal(t, len(records), 4) // header + 3 data rows

	// Test getting a specific record
	record, err := csvDoc.GetRecord(1)
	assert.Nil(t, err)
	assert.Equal(t, len(record), 3)
	assert.Equal(t, record[0], "John")
	assert.Equal(t, record[1], "25")
	assert.Equal(t, record[2], "NYC")

	// Test getting a specific cell
	cell, err := csvDoc.GetCell(2, 0)
	assert.Nil(t, err)
	assert.Equal(t, cell, "Jane")

	// Test record and column counts
	recordCount, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, recordCount, 3)

	colCount, err := csvDoc.ColumnCount()
	assert.Nil(t, err)
	assert.Equal(t, colCount, 3)
}

func TestCSVDocument_UpdateCell(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Create a test CSV file
	csvContent := "name,age,city\nJohn,25,NYC\nJane,30,LA\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Update a cell
	err = csvDoc.UpdateCell(1, 0, "Johnny")
	assert.Nil(t, err)

	// Verify the update by reading the cell back
	cell, err := csvDoc.GetCell(1, 0)
	assert.Nil(t, err)
	assert.Equal(t, cell, "Johnny")

	// Verify the file was actually saved
	content, err := doc.Content()
	assert.Nil(t, err)
	assert.True(t, containsString(content, "Johnny"))
	assert.False(t, containsString(content, "John,25")) // Old value should be gone
}

func TestCSVDocument_UpdateRecord(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	csvContent := "name,age,city\nJohn,25,NYC\nJane,30,LA\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Update an entire record
	newRecord := []string{"Johnny", "26", "Boston"}
	err = csvDoc.UpdateRecord(1, newRecord)
	assert.Nil(t, err)

	// Verify the update
	record, err := csvDoc.GetRecord(1)
	assert.Nil(t, err)
	assert.Equal(t, record[0], "Johnny")
	assert.Equal(t, record[1], "26")
	assert.Equal(t, record[2], "Boston")
}

func TestCSVDocument_AddRecord(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	csvContent := "name,age,city\nJohn,25,NYC\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Get initial record count
	initialCount, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, initialCount, 1)

	// Add a new record
	newRecord := []string{"Alice", "28", "Seattle"}
	err = csvDoc.AddRecord(newRecord)
	assert.Nil(t, err)

	// Verify the record was added
	count, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, count, 2)

	// Verify the new record
	record, err := csvDoc.GetRecord(2)
	assert.Nil(t, err)
	assert.Equal(t, record[0], "Alice")
	assert.Equal(t, record[1], "28")
	assert.Equal(t, record[2], "Seattle")
}

func TestCSVDocument_DeleteRecord(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	csvContent := "name,age,city\nJohn,25,NYC\nJane,30,LA\nBob,35,Chicago\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Get initial count
	initialCount, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, initialCount, 3)

	// Delete the middle record (Jane)
	err = csvDoc.DeleteRecord(2)
	assert.Nil(t, err)

	// Verify count decreased
	count, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, count, 2)

	// Verify Jane's record is gone and Bob moved up
	record, err := csvDoc.GetRecord(2)
	assert.Nil(t, err)
	assert.Equal(t, record[0], "Bob")
}

func TestCSVDocument_Metadata(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	csvContent := "name,age,city\nJohn,25,NYC\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Test getting default metadata (should not fail)
	metadata, err := csvDoc.GetMetadata()
	assert.Nil(t, err)
	assert.NotNil(t, metadata)
	assert.Equal(t, metadata.Title, "") // Should be empty initially

	// Test saving metadata
	testMetadata := &files.CSVMetadata{
		Title:       "Test Data",
		Description: "Sample CSV for testing",
		SortColumn:  "name",
		Headers:     []string{"Name", "Age", "City"},
	}

	err = csvDoc.SaveMetadata(testMetadata)
	assert.Nil(t, err)

	// Verify metadata file was created
	assert.True(t, rm.FileExists("test.csv.meta.json"))

	// Test loading metadata back
	loadedMeta, err := csvDoc.GetMetadata()
	assert.Nil(t, err)
	assert.Equal(t, loadedMeta.Title, "Test Data")
	assert.Equal(t, loadedMeta.Description, "Sample CSV for testing")
	assert.Equal(t, loadedMeta.SortColumn, "name")
	assert.Equal(t, len(loadedMeta.Headers), 3)
	assert.Equal(t, loadedMeta.Headers[0], "Name")
}

func TestCSVDocument_ErrorHandling(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	csvContent := "name,age\nJohn,25\n"
	err = rm.WriteString("test.csv", csvContent)
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("test.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Test out of bounds record access
	_, err = csvDoc.GetRecord(5)
	assert.NotNil(t, err)
	assert.True(t, containsString(err.Error(), "out of range"))

	// Test out of bounds column access
	_, err = csvDoc.GetCell(1, 5)
	assert.NotNil(t, err)
	assert.True(t, containsString(err.Error(), "out of range"))

	// Test invalid update
	err = csvDoc.UpdateCell(5, 0, "test")
	assert.NotNil(t, err)
	assert.True(t, containsString(err.Error(), "out of range"))

	// Test invalid delete
	err = csvDoc.DeleteRecord(5)
	assert.NotNil(t, err)
	assert.True(t, containsString(err.Error(), "out of range"))
}

func TestCSVDocument_EmptyFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Create empty CSV file
	err = rm.WriteString("empty.csv", "")
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("empty.csv")
	assert.Nil(t, err)

	csvDoc := files.NewCSVDocument(doc)

	// Test operations on empty file
	count, err := csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, count, 0)

	colCount, err := csvDoc.ColumnCount()
	assert.Nil(t, err)
	assert.Equal(t, colCount, 0)

	// Test adding first record
	newRecord := []string{"name", "age"}
	err = csvDoc.AddRecord(newRecord)
	assert.Nil(t, err)

	count, err = csvDoc.RecordCount()
	assert.Nil(t, err)
	assert.Equal(t, count, 0)
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
