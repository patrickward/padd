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

	return padd.NewFileRepository(rm, padd.DefaultFileConfig), rm
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

	fr.ReloadCaches()
	assert.True(t, fr.FilePathExists("inbox.md"))
	assert.True(t, fr.FilePathExists("active.md"))
	assert.True(t, fr.FilePathExists("resources/looney.md"))
	assert.True(t, fr.FilePathExists("resources/characters/roadrunner.md"))
}

func TestFileRepository_ReloadResource(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, rm := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)
	fr.ReloadCaches()

	// Add a new resource directly for this test
	err = rm.WriteString("resources/foobar.md", "test")
	assert.Nil(t, err)
	assert.True(t, fr.FilePathExists("resources/foobar.md"))

	// Resource shouldn't exist in the cache yet
	_, err = fr.FileInfo("resources/foobar")
	assert.NotNil(t, err)

	// Reload the resource and check the cache
	fr.ReloadResource("resources/foobar.md")
	assert.True(t, fr.FilePathExists("resources/foobar.md"))
	info, err := fr.FileInfo("resources/foobar")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/foobar")
	assert.Equal(t, info.Path, "resources/foobar.md")
	assert.Equal(t, info.Display, "Foobar")
	assert.Equal(t, info.DisplayBase, "Foobar")
}

func TestFileRepository_FileInfo(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	file, err := fr.FileInfo("inbox")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "inbox")
	assert.Equal(t, file.Path, "inbox.md")
	assert.Equal(t, file.Display, "Inbox")

	file, err = fr.FileInfo("active")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "active")
	assert.Equal(t, file.Path, "active.md")
	assert.Equal(t, file.Display, "Active")

	file, err = fr.FileInfo("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "resources/looney")
	assert.Equal(t, file.Path, "resources/looney.md")
	assert.Equal(t, file.Display, "Looney")

	file, err = fr.FileInfo("resources/characters/wile")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "resources/characters/wile")
	assert.Equal(t, file.Path, "resources/characters/wile.md")
	assert.Equal(t, file.Display, "Characters/Wile")

	file, err = fr.FileInfo("daily/2025/09-september")
	assert.Nil(t, err)
	assert.Equal(t, file.ID, "daily/2025/09-september")
	assert.Equal(t, file.Path, "daily/2025/09-september.md")
	assert.Equal(t, file.Display, "September 2025")
	assert.Equal(t, file.DisplayBase, "September 2025")
	assert.Equal(t, file.IsTemporal, true)
	assert.Equal(t, file.Directory, "daily/2025")
	assert.Equal(t, file.Year, "2025")
	assert.Equal(t, file.Month, "09")
	assert.Equal(t, file.MonthName, "September")

	_, err = fr.FileInfo("nonexistent")
	assert.NotNil(t, err)
}

func TestFileRepository_CoreFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	coreFiles := fr.CoreFiles()
	assert.Equal(t, len(fr.CoreFiles()), 2)
	assert.Equal(t, coreFiles["inbox"].Path, "inbox.md")
	assert.Equal(t, coreFiles["inbox"].Display, "Inbox")
	assert.Equal(t, coreFiles["active"].Path, "active.md")
	assert.Equal(t, coreFiles["active"].Display, "Active")
}

func TestFileRepository_FileInfo_CoreFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	info, err := fr.FileInfo("inbox")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "inbox")
	assert.Equal(t, info.Path, "inbox.md")
	assert.Equal(t, info.Display, "Inbox")

	_, err = fr.FileInfo("nonexistent.md")
	assert.NotNil(t, err)
}

func TestFileRepository_FileInfo_ResourceFiles(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()
	info, err := fr.FileInfo("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/looney")
	assert.Equal(t, info.Path, "resources/looney.md")
	assert.Equal(t, info.Display, "Looney")
	assert.Equal(t, info.DisplayBase, "Looney")

	info, err = fr.FileInfo("resources/characters/wile")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "resources/characters/wile")
	assert.Equal(t, info.Path, "resources/characters/wile.md")
	assert.Equal(t, info.Display, "Characters/Wile")
	assert.Equal(t, info.DisplayBase, "Wile")

	_, err = fr.FileInfo("resources/nonexistent.md")
	assert.NotNil(t, err)
}

func TestFileRepository_ResourcesDirectory(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	dir := fr.ResourcesTree()

	assert.Equal(t, len(dir.Files), 1)
	assert.Equal(t, dir.Files[0].Path, "resources/looney.md")
	assert.Equal(t, len(dir.Directories), 1)
	assert.Equal(t, dir.Directories["characters"].Files[0].Path, "resources/characters/roadrunner.md")
	assert.Equal(t, dir.Directories["characters"].Files[1].Path, "resources/characters/wile.md")
	assert.Equal(t, len(dir.Directories["characters"].Directories), 1)
	assert.Equal(t, dir.Directories["characters"].Directories["minor"].Files[0].Path, "resources/characters/minor/michigan.md")
}

func TestFileRepository_TemporalTree(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	dirs := []string{"daily", "journal"}
	for _, dir := range dirs {
		years, files, err := fr.TemporalTree(dir)
		assert.Nil(t, err)

		assert.Equal(t, len(years), 2)
		// Newest year first
		assert.Equal(t, years[0], "2025")
		assert.Equal(t, years[1], "2024")
		assert.Equal(t, len(files["2025"]), 2)
		assert.Equal(t, files["2025"][0].Path, dir+"/2025/09-september.md")
		assert.Equal(t, files["2025"][1].Path, dir+"/2025/08-august.md")
		assert.Equal(t, len(files["2024"]), 2)
		assert.Equal(t, files["2024"][0].Path, dir+"/2024/12-december.md")
		assert.Equal(t, files["2024"][1].Path, dir+"/2024/11-november.md")
	}
}

func TestFileRepository_GetTemporalFile(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	// Create time var for September 5, 2025
	current := time.Date(2025, time.September, 5, 0, 0, 0, 0, time.UTC)

	file, found := fr.TemporalFileInfo("daily", current)
	assert.True(t, found)
	assert.Equal(t, file.Path, "daily/2025/09-september.md")
	assert.Equal(t, file.Display, "September 2025")
	assert.Equal(t, file.DisplayBase, "September 2025")
	assert.Equal(t, file.IsTemporal, true)
	assert.Equal(t, file.Directory, "daily/2025")
	assert.Equal(t, file.Year, "2025")
	assert.Equal(t, file.Month, "09")
	assert.Equal(t, file.MonthName, "September")
}

func TestFileRepository_GetDocument(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	doc, err := fr.GetDocument("inbox")
	assert.Nil(t, err)

	assert.Equal(t, doc.Info.Path, "inbox.md")
	assert.Equal(t, doc.Info.Display, "Inbox")

	doc, err = fr.GetDocument("resources/looney")
	assert.Nil(t, err)
	assert.Equal(t, doc.Info.Path, "resources/looney.md")
	assert.Equal(t, doc.Info.Display, "Looney")
	assert.Equal(t, doc.Info.DisplayBase, "Looney")

	doc, err = fr.GetDocument("nonexistent")
	assert.NotNil(t, err)
}

func TestFileRepository_GetOrCreateResourceDocument(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	fr.ReloadCaches()

	assert.False(t, fr.FilePathExists("resources/foobar.md"))

	doc, err := fr.GetOrCreateResourceDocument("foobar")
	assert.Nil(t, err)
	assert.True(t, fr.FilePathExists("resources/foobar.md"))
	assert.Equal(t, doc.Info.Path, "resources/foobar.md")
	assert.Equal(t, doc.Info.Display, "Foobar")
	assert.Equal(t, doc.Info.DisplayBase, "Foobar")
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
	assert.Equal(t, doc.Info.Display, "March 2025")
	assert.Equal(t, doc.Info.DisplayBase, "March 2025")
	assert.Equal(t, doc.Info.IsTemporal, true)
	assert.Equal(t, doc.Info.Directory, "daily/2025")
	assert.Equal(t, doc.Info.Year, "2025")
	assert.Equal(t, doc.Info.Month, "03")
	assert.Equal(t, doc.Info.MonthName, "March")
	assert.Equal(t, doc.Info.ID, "daily/2025/03-march")
	assert.Equal(t, doc.Info.Path, "daily/2025/03-march.md")
}
