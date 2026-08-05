package settings

import (
	"context"
	"path/filepath"
	"testing"

	"easypour/service/internal/cvar"
	"easypour/service/internal/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_GetReturnsEmptyByDefault(t *testing.T) {
	store := openTestStore(t)
	s, err := store.Get()
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.Empty(t, s.AppriseURL)
}

func TestStore_UpdateAndGetAppriseURL(t *testing.T) {
	store := openTestStore(t)
	err := store.Update(&Settings{AppriseURL: "http://localhost:8000/notify/"})
	require.NoError(t, err)

	s, err := store.Get()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8000/notify/", s.AppriseURL)
}

func TestStore_UpdateClearsAppriseURL(t *testing.T) {
	store := openTestStore(t)
	require.NoError(t, store.Update(&Settings{AppriseURL: "http://example/notify"}))
	require.NoError(t, store.Update(&Settings{AppriseURL: ""}))

	s, err := store.Get()
	require.NoError(t, err)
	assert.Empty(t, s.AppriseURL)
}

func TestStore_UpdateTrimsWhitespace(t *testing.T) {
	store := openTestStore(t)
	require.NoError(t, store.Update(&Settings{AppriseURL: "  http://localhost:8000/notify/  "}))

	s, err := store.Get()
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:8000/notify/", s.AppriseURL)
}

func TestStore_EnsureDefaultCvars_InsertsAndPreservesValues(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.EnsureDefaultCvars(ctx, "EasyPour"))

	rows, err := store.ListCvars(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	require.NoError(t, store.UpdateCvar(ctx, cvar.KeySiteTitle, 0, "Custom Title"))
	require.NoError(t, store.EnsureDefaultCvars(ctx, "EasyPour"))

	row, err := store.FindCvar(ctx, cvar.KeySiteTitle)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "Custom Title", row.ValueString)
	assert.Equal(t, "Site title", row.Title)
}

func TestStore_EnsureDefaultCvars_MigratesLegacyApprise(t *testing.T) {
	store := openTestStoreWithoutDefaults(t)
	_, err := store.db.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)`, cvar.KeyAppriseURL, "http://legacy/notify")
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.EnsureDefaultCvars(ctx, "EasyPour"))

	row, err := store.FindCvar(ctx, cvar.KeyAppriseURL)
	require.NoError(t, err)
	require.NotNil(t, row)
	assert.Equal(t, "http://legacy/notify", row.ValueString)
}

func TestStore_InsertCvarIfMissing_RefreshesMetadataOnly(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.UpdateCvar(ctx, cvar.KeySiteTitle, 0, "Kept"))

	require.NoError(t, store.InsertCvarIfMissing(ctx, CvarRow{
		Key: cvar.KeySiteTitle, MainType: cvar.TypeString, ValueString: "ShouldNotOverwrite",
		Title: "Updated title", Description: "Updated desc", Category: "Site", Ordinal: 10,
	}))

	row, err := store.FindCvar(ctx, cvar.KeySiteTitle)
	require.NoError(t, err)
	assert.Equal(t, "Kept", row.ValueString)
	assert.Equal(t, "Updated title", row.Title)
	assert.Equal(t, "Updated desc", row.Description)
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store := openTestStoreWithoutDefaults(t)
	require.NoError(t, store.EnsureDefaultCvars(context.Background(), "EasyPour"))
	return store
}

func openTestStoreWithoutDefaults(t *testing.T) *Store {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.ApplySchema(db))
	return NewStore(db)
}
