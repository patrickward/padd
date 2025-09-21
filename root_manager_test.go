package padd_test

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/patrickward/padd"
	"github.com/patrickward/padd/internal/assert"
)

const defaultDataDir = "./testdata/data"

func setupRootManager(t *testing.T, dataDir string) *padd.RootManager {
	t.Helper()

	rm, err := padd.NewRootManager(dataDir)
	assert.Nil(t, err)

	return rm
}

func TestRootManager_ReadFile(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)
	content, err := rm.ReadFile("inbox.md")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(content), "title: Inbox"))

	content, err = rm.ReadFile("resources/looney.md")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(content), "title: Looney Tunes"))

	_, err = rm.ReadFile("nonexistent.md")
	assert.NotNil(t, err)
}

func TestRootManager_WriteFile(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.WriteFile("test.md", []byte("testy test"), 0644)
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test.md"))
	content, err := rm.ReadFile("test.md")
	assert.Nil(t, err)
	assert.Equal(t, string(content), "testy test")
}

func TestRootManager_WriteString(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.WriteString("test.md", "testy test")
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test.md"))
	content, err := rm.ReadFile("test.md")
	assert.Nil(t, err)
	assert.Equal(t, string(content), "testy test")
}

func TestRootManager_Stat(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)
	info, err := rm.Stat("inbox.md")
	assert.Nil(t, err)
	assert.Equal(t, info.Name(), "inbox.md")
	assert.True(t, info.Size() > 0)
	assert.Equal(t, info.IsDir(), false)
	assert.Equal(t, info.Mode(), 0644)
	assert.NotNil(t, info.ModTime())
	assert.Equal(t, info.ModTime().IsZero(), false)
}

func TestRootManager_MkdirAll(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.MkdirAll("test/test2/test3", 0755)
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test/test2/test3"))
}

func TestRootManager_Remove(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.WriteFile("test.md", []byte("testy test"), 0644)
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test.md"))
	err = rm.Remove("test.md")
	assert.Nil(t, err)
	assert.False(t, rm.FileExists("test.md"))
}

func TestRootManager_RemoveAll(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.MkdirAll("test/", 0755)
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test/"))
	err = rm.WriteFile("test/test1.md", []byte("testy once"), 0644)
	assert.Nil(t, err)
	err = rm.WriteFile("test/test2.md", []byte("testy twice"), 0644)
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test/test1.md"))
	assert.True(t, rm.FileExists("test/test2.md"))
	err = rm.RemoveAll("test/")
	assert.Nil(t, err)
	assert.False(t, rm.FileExists("test/"))
	assert.False(t, rm.FileExists("test/test1.md"))
	assert.False(t, rm.FileExists("test/test2.md"))
}

func TestRootManager_FileExists(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)
	assert.True(t, rm.FileExists("inbox.md"))
	assert.False(t, rm.FileExists("nonexistent.md"))
}

func TestRootManager_WalkDir(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)
	files := make([]string, 0)
	dirs := make([]string, 0)
	err := rm.WalkDir("resources", func(path string, info fs.DirEntry, err error) error {
		if info.IsDir() {
			dirs = append(dirs, path)
		} else {
			files = append(files, path)
		}

		return nil
	})
	assert.Nil(t, err)

	assert.Equal(t, len(files), 4)
	assert.Equal(t, len(dirs), 3)
}

func TestRootManager_Scan(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)

	result, err := rm.Scan("resources", func(s string, entry fs.DirEntry) bool {
		// Filter out directories
		return !entry.IsDir()
	})

	assert.Nil(t, err)
	assert.Equal(t, len(result), 4)
}

func TestRootManager_ReadDir(t *testing.T) {
	rm := setupRootManager(t, defaultDataDir)
	files, err := rm.ReadDir("resources")
	assert.Nil(t, err)

	// 1 directory and 1 file
	assert.Equal(t, len(files), 2)
	assert.Equal(t, files[0].Name(), "characters")
	assert.Equal(t, files[1].Name(), "looney.md")
}

func TestRootManager_CreateFileIfNotExists(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	err := rm.CreateFileIfNotExists("test.md", "testy test")
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test.md"))
	content, err := rm.ReadFile("test.md")
	assert.Nil(t, err)
	assert.Equal(t, string(content), "testy test")
}

func TestRootManager_CreateDirectoryIfNotExists(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)
	assert.False(t, rm.FileExists("test"))
	assert.False(t, rm.FileExists("test/test-nested"))
	err := rm.CreateDirectoryIfNotExists("test/test-nested")
	assert.Nil(t, err)
	assert.True(t, rm.FileExists("test"))
	assert.True(t, rm.FileExists("test/test-nested"))
}

func TestRootManager_ResolveMonthlyFile(t *testing.T) {
	tmp := t.TempDir()
	rm := setupRootManager(t, tmp)

	file, err := rm.ResolveMonthlyFile(time.Date(2025, time.September, 5, 0, 0, 0, 0, time.UTC), "daily")
	assert.Nil(t, err)
	assert.Equal(t, file, "daily/2025/09-september.md")

	file, err = rm.ResolveMonthlyFile(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC), "daily")
	assert.Nil(t, err)
	assert.Equal(t, file, "daily/2000/01-january.md")
}
