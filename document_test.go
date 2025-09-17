package padd_test

import (
	"strings"
	"testing"

	"github.com/patrickward/padd/assert"
)

func TestDocument_Content(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	doc, err := fr.GetDocument("inbox")
	if err != nil {
		t.Fatal(err)
	}

	content, err := doc.Content()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(content, "Test data for inbox"))
}

func TestDocument_Content_Resource(t *testing.T) {
	t.Parallel()
	fr, _ := setupTestFileRepo(t, "")
	fr.ReloadCaches()

	doc, err := fr.GetDocument("resources/looney")
	if err != nil {
		t.Fatal(err)
	}

	content, err := doc.Content()
	assert.Nil(t, err)
	assert.True(t, strings.Contains(content, "Looney Tunes is a classic animated series produced by Warner Bros."))
}

func TestDocument_Save(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	err := fr.Initialize()
	assert.Nil(t, err)
	fr.ReloadCaches()

	doc, err := fr.GetDocument("inbox")
	if err != nil {
		t.Fatal(err)
	}

	err = doc.Save("New content for the inbox")
	assert.Nil(t, err)

	// Reload the document and check the content
	doc, err = fr.GetDocument("inbox")
	if err != nil {
		t.Fatal(err)
	}

	content, err := doc.Content()
	assert.Nil(t, err)
	assert.Equal(t, content, "New content for the inbox")
}

func TestDocument_Save_NewFile(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	fr.ReloadCaches()

	err := fr.Initialize()
	assert.Nil(t, err)

	doc, err := fr.GetOrCreateResourceDocument("new-resource")
	if err != nil {
		t.Fatal(err)
	}

	err = doc.Save("New content for the new resource")
	assert.Nil(t, err)
	assert.True(t, fr.FilePathExists("resources/new-resource.md"))

	content, err := doc.Content()
	assert.Nil(t, err)
	assert.Equal(t, content, "New content for the new resource")
}

func TestDocument_Delete(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	fr, _ := setupTestFileRepo(t, tmp)
	fr.ReloadCaches()

	err := fr.Initialize()
	assert.Nil(t, err)

	assert.False(t, fr.FilePathExists("resources/new-resource.md"))

	doc, err := fr.GetOrCreateResourceDocument("new-resource")
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, fr.FilePathExists("resources/new-resource.md"))
	err = doc.Delete()
	assert.Nil(t, err)
	assert.False(t, fr.FilePathExists("resources/new-resource.md"))
}
