package settings

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSettingsDBPath_UsesConfigDir(t *testing.T) {
	path := GetSettingsDBPath("/data/config.yaml")
	assert.Equal(t, filepath.Join("/data", "settings.db"), path)
}

func TestGetSettingsDBPath_DefaultWhenEmpty(t *testing.T) {
	assert.Equal(t, "settings.db", GetSettingsDBPath(""))
}

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

func openTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "settings.db")
	store, err := NewStore(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())
	return store
}
