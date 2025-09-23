package files

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// CSVDocument represents a CSV document
type CSVDocument struct {
	*Document
	metadata  *CSVMetadata
	records   [][]string
	metaMu    sync.RWMutex
	recordsMu sync.RWMutex
}

// CSVMetadata represents the metadata for a CSV document
type CSVMetadata struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	SortColumn  string            `json:"sort_column,omitempty"`
	SortDesc    bool              `json:"sort_desc,omitempty"`
	ColumnTypes map[int]CellType  `json:"column_types,omitempty"`
	Headers     []string          `json:"headers,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

// CellType represents the type of a cell in a CSV document
type CellType string

const (
	CellTypeText CellType = "text"
	CellTypeDate CellType = "date"
	CellTypeTime CellType = "time"
	CellTypeBool CellType = "bool"
	CellTypeNum  CellType = "number"
)

// NewCSVDocument creates a new CSV document
func NewCSVDocument(doc *Document) *CSVDocument {
	return &CSVDocument{
		Document: doc,
		metadata: nil,
	}
}

func (c *CSVDocument) GetMetadata() (*CSVMetadata, error) {
	c.metaMu.RLock()
	if c.metadata != nil {
		defer c.metaMu.RUnlock()
		return c.metadata, nil
	}

	c.metaMu.RUnlock()

	return c.loadMetadata()
}

func (c *CSVDocument) SaveMetadata(metadata *CSVMetadata) error {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	metaPath := c.getMetadataPath()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal csv metadata: %w", err)
	}

	if err := c.repo.rootManager.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save csv metadata: %w", err)
	}

	c.metadata = metadata
	return nil
}

func (c *CSVDocument) GetRecords() ([][]string, error) {
	c.recordsMu.RLock()
	if c.records != nil {
		defer c.recordsMu.RUnlock()
		return c.records, nil
	}
	c.recordsMu.RUnlock()

	return c.loadRecords()
}

func (c *CSVDocument) GetRecord(row int) ([]string, error) {
	records, err := c.GetRecords()
	if err != nil {
		return nil, err
	}

	if row < 0 || row >= len(records) {
		return nil, fmt.Errorf("record index %d out of range (document has %d records)", row, len(records))
	}

	return records[row], nil
}

func (c *CSVDocument) GetCell(row, col int) (string, error) {
	record, err := c.GetRecord(row)
	if err != nil {
		return "", err
	}

	if col < 0 || col >= len(record) {
		return "", fmt.Errorf("column index %d out of range (record has %d columns)", col, len(record))
	}

	return record[col], nil
}

func (c *CSVDocument) UpdateCell(row, col int, value string) error {
	records, err := c.GetRecords()
	if err != nil {
		return err
	}

	if row < 0 || row >= len(records) {
		return fmt.Errorf("record index %d out of range (document has %d records)", row, len(records))
	}

	if col < 0 || col >= len(records[row]) {
		return fmt.Errorf("column index %d out of range (record has %d columns)", col, len(records[row]))
	}

	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()

	records[row][col] = value
	c.records = records

	return c.saveRecords(records)
}

func (c *CSVDocument) UpdateRecord(row int, values []string) error {
	records, err := c.GetRecords()
	if err != nil {
		return err
	}

	if row < 0 || row >= len(records) {
		return fmt.Errorf("record index %d out of range (document has %d records)", row, len(records))
	}

	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()

	records[row] = values
	c.records = records

	return c.saveRecords(records)
}

func (c *CSVDocument) AddRecord(values []string) error {
	records, err := c.GetRecords()
	if err != nil {
		return err
	}

	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()

	records = append(records, values)
	c.records = records

	return c.saveRecords(records)
}

func (c *CSVDocument) DeleteRecord(row int) error {
	records, err := c.GetRecords()
	if err != nil {
		return err
	}

	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()

	if row < 0 || row >= len(records) {
		return fmt.Errorf("record index %d out of range (document has %d records)", row, len(records))
	}

	records = append(records[:row], records[row+1:]...)
	c.records = records

	return c.saveRecords(records)
}

func (c *CSVDocument) RecordCount() (int, error) {
	records, err := c.GetRecords()
	if err != nil {
		return 0, err
	}

	if len(records) <= 0 {
		return 0, nil
	}

	return len(records) - 1, nil
}

func (c *CSVDocument) ColumnCount() (int, error) {
	records, err := c.GetRecords()
	if err != nil {
		return 0, err
	}

	if len(records) == 0 {
		return 0, nil
	}

	return len(records[0]), nil
}

func emptyCSVMetadata() *CSVMetadata {
	return &CSVMetadata{
		ColumnTypes: make(map[int]CellType),
		Headers:     make([]string, 0),
		Custom:      make(map[string]string),
	}
}

func (c *CSVDocument) loadMetadata() (*CSVMetadata, error) {
	c.metaMu.Lock()
	defer c.metaMu.Unlock()

	metaPath := c.getMetadataPath()

	if !c.repo.rootManager.FileExists(metaPath) {
		return emptyCSVMetadata(), nil
	}

	data, err := c.repo.rootManager.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load csv metadata: %w", err)
	}

	var metadata CSVMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal csv metadata: %w", err)
	}

	if metadata.ColumnTypes == nil {
		metadata.ColumnTypes = make(map[int]CellType)
	}

	if metadata.Headers == nil {
		metadata.Headers = make([]string, 0)
	}

	if metadata.Custom == nil {
		metadata.Custom = make(map[string]string)
	}

	return &metadata, nil
}

func (c *CSVDocument) getMetadataPath() string {
	return c.Info.Path + ".meta.json"
}

func (c *CSVDocument) loadRecords() ([][]string, error) {
	c.recordsMu.Lock()
	defer c.recordsMu.Unlock()

	content, err := c.Document.Content()
	if err != nil {
		return nil, fmt.Errorf("failed to load csv content: %w", err)
	}

	reader := csv.NewReader(strings.NewReader(content))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse csv content: %w", err)
	}

	c.records = records
	return records, nil
}

func (c *CSVDocument) saveRecords(records [][]string) error {
	var buf strings.Builder
	writer := csv.NewWriter(&buf)

	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write csv record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("failed to flush csv writer: %w", err)
	}

	return c.Document.Save(buf.String())
}
