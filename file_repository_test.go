package padd_test

import (
	"testing"
	"time"

	"github.com/patrickward/padd"

	"github.com/patrickward/padd/assert"
)

func setupFileRepository(t *testing.T) (*padd.FileRepository, *padd.RootManager) {
	t.Helper()
	rm, err := padd.NewRootManager("./testdata/data")
	assert.Nil(t, err)

	return padd.NewFileRepository(rm, padd.DefaultFileConfig), rm
}

func TestFileRepository_Initialize(t *testing.T) {
	tmp := t.TempDir()
	rm, err := padd.NewRootManager(tmp)
	assert.Nil(t, err)

	fr := padd.NewFileRepository(rm, padd.DefaultFileConfig)
	err = fr.Initialize()
	assert.Nil(t, err)

	// Ensure file exists
	assert.True(t, rm.FileExists("inbox.md"))
	assert.True(t, rm.FileExists("active.md"))
}

func TestFileRepository_FileInfo(t *testing.T) {
	fr, _ := setupFileRepository(t)
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

	fr, _ := setupFileRepository(t)
	coreFiles := fr.CoreFiles()
	assert.Equal(t, len(fr.CoreFiles()), 2)
	assert.Equal(t, coreFiles["inbox"].Path, "inbox.md")
	assert.Equal(t, coreFiles["inbox"].Display, "Inbox")
	assert.Equal(t, coreFiles["active"].Path, "active.md")
	assert.Equal(t, coreFiles["active"].Display, "Active")
}

func TestFileRepository_FileInfo_CoreFiles(t *testing.T) {
	fr, _ := setupFileRepository(t)
	info, err := fr.FileInfo("inbox")
	assert.Nil(t, err)
	assert.Equal(t, info.ID, "inbox")
	assert.Equal(t, info.Path, "inbox.md")
	assert.Equal(t, info.Display, "Inbox")

	_, err = fr.FileInfo("nonexistent.md")
	assert.NotNil(t, err)
}

func TestFileRepository_FileInfo_ResourceFiles(t *testing.T) {
	fr, _ := setupFileRepository(t)
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
	fr, _ := setupFileRepository(t)
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
	fr, _ := setupFileRepository(t)
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
	fr, _ := setupFileRepository(t)
	fr.ReloadCaches()

	// Create time var for September 5, 2025
	current := time.Date(2025, time.September, 5, 0, 0, 0, 0, time.UTC)

	file, err := fr.TemporalFileInfo("daily", current)
	assert.Nil(t, err)
	assert.Equal(t, file.Path, "daily/2025/09-september.md")
	assert.Equal(t, file.Display, "September 2025")
}
