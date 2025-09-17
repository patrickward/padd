package padd_test

import (
	"testing"
	"time"

	"github.com/patrickward/padd"

	"github.com/patrickward/padd/assert"
)

func setupTestFileRepo(t *testing.T, path string) (*padd.FileRepository, *padd.RootManager) {
	t.Helper()

	if path == "" {
		path = "./testdata/data"
	}

	rm, err := padd.NewRootManager(path)
	assert.Nil(t, err)

	fr := padd.NewFileRepository(rm, padd.DefaultFileConfig)
	fr.ReloadCaches()

	return fr, rm
}

func TestFileRepository_Initialize(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Ensure file exists
	assert.True(t, rm.FileExists("inbox.md"))
	assert.True(t, rm.FileExists("active.md"))
}

func TestFileRepository_ReloadCaches(t *testing.T) {
	fr, _ := setupTestFileRepo(t, "")
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches, now that we've initialized
	fr.ReloadCaches()

	assert.True(t, fr.FilePathExists("inbox.md"))
	assert.True(t, fr.FilePathExists("active.md"))
	assert.True(t, fr.FilePathExists("resources/looney.md"))
	assert.True(t, fr.FilePathExists("resources/characters/roadrunner.md"))
}

func TestFileRepository_ReloadResources(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	// Explicitly reload caches, now that we've initialized'
	fr.ReloadCaches()

	// Add a new resource directly for this test
	err = rm.WriteString("resources/foobar.md", "test")
	assert.Nil(t, err)
	assert.True(t, fr.FilePathExists("resources/foobar.md"))

	// Resource shouldn't exist in the cache yet
	_, err = fr.FileInfo("resources/foobar")
	assert.NotNil(t, err)

	// Reload the resource and check the cache
	fr.ReloadResources()
	assert.True(t, fr.FilePathExists("resources/foobar.md"))
	info, err := fr.FileInfo("resources/foobar")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/foobar")
	assert.Equal(t, info.Path, "resources/foobar.md")
	assert.Equal(t, info.Title, "Foobar")
	assert.Equal(t, info.TitleBase, "Foobar")
}

func TestFileRepository_FileInfo(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	file, err := fr.FileInfo("inbox")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "inbox")
	assert.Equal(t, file.Path, "inbox.md")
	assert.Equal(t, file.Title, "Inbox")

	file, err = fr.FileInfo("active")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "active")
	assert.Equal(t, file.Path, "active.md")
	assert.Equal(t, file.Title, "Active")

	file, err = fr.FileInfo("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "resources/looney")
	assert.Equal(t, file.Path, "resources/looney.md")
	assert.Equal(t, file.Title, "Looney")

	file, err = fr.FileInfo("resources/characters/wile")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "resources/characters/wile")
	assert.Equal(t, file.Path, "resources/characters/wile.md")
	assert.Equal(t, file.Title, "Characters/Wile")

	file, err = fr.FileInfo("daily/2025/09-september")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "daily/2025/09-september")
	assert.Equal(t, file.Path, "daily/2025/09-september.md")
	assert.Equal(t, file.Title, "Daily/2025/09 September")
	assert.Equal(t, file.TitleBase, "09 September")
	assert.Equal(t, file.IsTemporal, true)
	assert.Equal(t, file.DirectoryPath, "daily/2025")
	assert.Equal(t, file.Year(), "2025")
	assert.Equal(t, file.Month(), "09")
	assert.Equal(t, file.MonthName(), "September")

	_, err = fr.FileInfo("nonexistent")
	assert.NotNil(t, err)
}

func TestFileRepository_CoreFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	coreFiles := fr.CoreFiles()
	assert.Equal(t, len(fr.CoreFiles()), 2)
	assert.Equal(t, coreFiles["inbox"].Path, "inbox.md")
	assert.Equal(t, coreFiles["inbox"].Title, "Inbox")
	assert.Equal(t, coreFiles["active"].Path, "active.md")
	assert.Equal(t, coreFiles["active"].Title, "Active")
}

func TestFileRepository_FileInfo_CoreFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	info, err := fr.FileInfo("inbox")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "inbox")
	assert.Equal(t, info.Path, "inbox.md")
	assert.Equal(t, info.Title, "Inbox")

	_, err = fr.FileInfo("nonexistent.md")
	assert.NotNil(t, err)
}

func TestFileRepository_FileInfo_ResourceFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	info, err := fr.FileInfo("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/looney")
	assert.Equal(t, info.Path, "resources/looney.md")
	assert.Equal(t, info.Title, "Looney")
	assert.Equal(t, info.TitleBase, "Looney")

	info, err = fr.FileInfo("resources/characters/wile")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/characters/wile")
	assert.Equal(t, info.Path, "resources/characters/wile.md")
	assert.Equal(t, info.Title, "Characters/Wile")
	assert.Equal(t, info.TitleBase, "Wile")

	_, err = fr.FileInfo("resources/nonexistent.md")
	assert.NotNil(t, err)
}

func TestFileRepository_DirectoryTreeFor(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	dir := fr.DirectoryTreeFor(fr.Config().ResourcesDirectory)

	assert.Equal(t, len(dir.Files), 1)
	assert.Equal(t, dir.Files[0].Path, "resources/looney.md")
	assert.Equal(t, len(dir.Directories), 1)
	assert.Equal(t, dir.Directories["characters"].Files[0].Path, "resources/characters/roadrunner.md")
	assert.Equal(t, dir.Directories["characters"].Files[1].Path, "resources/characters/wile.md")
	assert.Equal(t, len(dir.Directories["characters"].Directories), 1)
	assert.Equal(t, dir.Directories["characters"].Directories["minor"].Files[0].Path, "resources/characters/minor/michigan.md")
}

func TestFileRepository_GetTemporalFile(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	// Create time var for September 5, 2025
	current := time.Date(2025, time.September, 5, 0, 0, 0, 0, time.UTC)

	file, found := fr.TemporalFileInfo("daily", current)
	assert.True(t, found)
	assert.Equal(t, file.Path, "daily/2025/09-september.md")
	assert.Equal(t, file.Title, "September 2025")
	assert.Equal(t, file.TitleBase, "September 2025")
	assert.Equal(t, file.IsTemporal, true)
	assert.Equal(t, file.DirectoryPath, "daily/2025")
	assert.Equal(t, file.Year(), "2025")
	assert.Equal(t, file.Month(), "09")
	assert.Equal(t, file.MonthName(), "September")
}

func TestFileRepository_GetDocument(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")

	doc, err := fr.GetDocument("inbox")
	assert.Nil(t, err)

	assert.Equal(t, doc.Info.Path, "inbox.md")
	assert.Equal(t, doc.Info.Title, "Inbox")

	doc, err = fr.GetDocument("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, doc.Info.Path, "resources/looney.md")
	assert.Equal(t, doc.Info.Title, "Looney")
	assert.Equal(t, doc.Info.TitleBase, "Looney")

	_, err = fr.GetDocument("nonexistent")
	assert.NotNil(t, err)
}

func TestFileRepository_GetOrCreateResourceDocument(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)

	assert.False(t, fr.FilePathExists("resources/foobar.md"))

	doc, err := fr.GetOrCreateResourceDocument("foobar")
	assert.Nil(t, err)
	assert.True(t, fr.FilePathExists("resources/foobar.md"))
	assert.Equal(t, doc.Info.Path, "resources/foobar.md")
	assert.Equal(t, doc.Info.Title, "Foobar")
	assert.Equal(t, doc.Info.TitleBase, "Foobar")
	assert.Equal(t, doc.Info.ID, "resources/foobar")
	assert.Equal(t, doc.Info.Path, "resources/foobar.md")
}

func TestFileRepository_GetOrCreateTemporalDocument(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)

	doc, err := fr.GetOrCreateTemporalDocument("daily", time.Date(2025, time.March, 3, 0, 0, 0, 0, time.UTC))
	assert.Nil(t, err)
	assert.Equal(t, doc.Info.Path, "daily/2025/03-march.md")
	assert.Equal(t, doc.Info.Title, "March 2025")
	assert.Equal(t, doc.Info.TitleBase, "March 2025")
	assert.Equal(t, doc.Info.IsTemporal, true)
	assert.Equal(t, doc.Info.DirectoryPath, "daily/2025")
	assert.Equal(t, doc.Info.Year(), "2025")
	assert.Equal(t, doc.Info.Month(), "03")
	assert.Equal(t, doc.Info.MonthName(), "March")
	assert.Equal(t, doc.Info.ID, "daily/2025/03-march")
	assert.Equal(t, doc.Info.Path, "daily/2025/03-march.md")
}

func TestFileRepository_NormalizeFileName_EdgeCases(t *testing.T) {
	fr, _ := setupTestFileRepo(t, "")

	testCases := []struct {
		input    string
		expected string
	}{
		{"", "untitled"},
		{"   ", "untitled"},
		{"Hello World!.md", "hello-world"},
		{"file (1).md", "file-1"},
		{"my_file-name.md", "my-file-name"},
		{"resources/sub dir/file&name.md", "resources/sub-dir/file-name"},
		{"café.md", "caf"}, // International characters
		{"123.md", "123"},
		{"---test---.md", "test"},
		{"test//.md", "test"},
	}

	for _, tc := range testCases {
		result := fr.CreateID(tc.input + ".md")
		assert.Equal(t, result, tc.expected)
	}
}
