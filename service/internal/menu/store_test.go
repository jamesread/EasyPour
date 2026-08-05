package menu

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/internal/sqlite"
)

func TestGetMenuPath_UsesConfigDir(t *testing.T) {
	path := GetMenuPath("/data/config.yaml")
	assert.Equal(t, filepath.Join("/data", "menu.yaml"), path)
}

func TestStore_EmptyDBSeedsDefaults(t *testing.T) {
	store := openTestStore(t)
	items, err := store.Load()
	require.NoError(t, err)
	require.Len(t, items, 3)
	ids := map[string]bool{}
	for _, it := range items {
		ids[it.Id] = true
	}
	assert.True(t, ids["coffee"])
	assert.True(t, ids["tea"])
	assert.True(t, ids["hot-chocolate"])
}

func TestStore_CreateUpdateDelete(t *testing.T) {
	store := openTestStore(t)

	created, err := store.Create(&easypourv1.MenuItem{
		Name:          "Espresso",
		Description:   "Short shot",
		SupportsSugar: true,
		SupportsMilk:  false,
		Category:      "Drinks",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.Id)
	assert.Equal(t, "Espresso", created.Name)

	items, err := store.Load()
	require.NoError(t, err)
	assert.Len(t, items, 4)

	created.Name = "Double Espresso"
	updated, err := store.Update(created)
	require.NoError(t, err)
	assert.Equal(t, "Double Espresso", updated.Name)

	require.NoError(t, store.Delete(created.Id))
	items, err = store.Load()
	require.NoError(t, err)
	assert.Len(t, items, 3)

	_, err = store.Update(created)
	require.Error(t, err)
	err = store.Delete(created.Id)
	require.Error(t, err)
}

func TestStore_ImportsYAMLWhenEmpty(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "menu.yaml")
	err := os.WriteFile(yamlPath, []byte(`
items:
  - id: latte
    name: Latte
    description: Milk coffee
    supports_sugar: true
    supports_milk: true
    image_url: /images/latte.png
    category: Drinks
`), 0644)
	require.NoError(t, err)

	store := openTestStoreIn(t, dir)
	items, err := store.Load()
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "latte", items[0].Id)
	assert.Equal(t, "Latte", items[0].Name)
	assert.Equal(t, "/images/latte.png", items[0].ImageUrl)
	assert.True(t, items[0].SupportsSugar)
	assert.True(t, items[0].SupportsMilk)
}

func TestStore_SkipsYAMLImportWhenRowsExist(t *testing.T) {
	dir := t.TempDir()
	db, err := sqlite.Open(filepath.Join(dir, "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.ApplySchema(db))

	store := NewStore(db, dir)
	require.NoError(t, store.SeedIfEmpty())

	err = os.WriteFile(filepath.Join(dir, "menu.yaml"), []byte(`
items:
  - id: should-not-import
    name: Ignored
    description: x
    supports_sugar: false
    supports_milk: false
    category: Other
`), 0644)
	require.NoError(t, err)

	require.NoError(t, store.SeedIfEmpty())

	items, err := store.Load()
	require.NoError(t, err)
	require.Len(t, items, 3)
	for _, it := range items {
		assert.NotEqual(t, "should-not-import", it.Id)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	return openTestStoreIn(t, t.TempDir())
}

func openTestStoreIn(t *testing.T, dir string) *Store {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(dir, "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.ApplySchema(db))
	store := NewStore(db, dir)
	require.NoError(t, store.SeedIfEmpty())
	return store
}
