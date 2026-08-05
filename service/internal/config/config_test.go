package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredMigration(t *testing.T) {
	if RequiredMigration != "0.base.sql" {
		t.Fatalf("migration=%s", RequiredMigration)
	}
}

func TestListenAddr_DefaultsWhenPORTUnset(t *testing.T) {
	t.Setenv("PORT", "")
	assert.Equal(t, ":9654", ListenAddr())
}

func TestListenAddr_UsesBarePORT(t *testing.T) {
	t.Setenv("PORT", "8080")
	assert.Equal(t, ":8080", ListenAddr())
}

func TestListenAddr_UsesFullAddressPORT(t *testing.T) {
	t.Setenv("PORT", "0.0.0.0:9000")
	assert.Equal(t, "0.0.0.0:9000", ListenAddr())
}

func TestListenAddr_TrimsWhitespace(t *testing.T) {
	t.Setenv("PORT", "  3000  ")
	assert.Equal(t, ":3000", ListenAddr())
}

func TestGetConfigPath_ReturnsEmptyWhenNoFileExists(t *testing.T) {
	orig := os.Getenv("EASYPOUR_CONFIG_FILE")
	defer os.Setenv("EASYPOUR_CONFIG_FILE", orig)
	os.Unsetenv("EASYPOUR_CONFIG_FILE")

	path := GetConfigPath()
	assert.Empty(t, path)
}

func TestLoadConfig_ReturnsDefaultsWhenNoFile(t *testing.T) {
	cfg := LoadConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.ConfigVersion)
	assert.Nil(t, cfg.Auth)
	assert.Empty(t, cfg.Webhooks)
	assert.Empty(t, cfg.OAuthProviders)
}

func TestLoadConfig_LoadsYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(`
configVersion: 1
webhooks:
  - url: "https://example.com/hook"
oauthProviders:
  - id: "google"
    name: "Google"
    auth_url: "https://example.com/auth"
`), 0644)
	require.NoError(t, err)

	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	cfg := LoadConfig()
	require.NotNil(t, cfg)
	assert.Equal(t, 1, cfg.ConfigVersion)
	require.Len(t, cfg.Webhooks, 1)
	assert.Equal(t, "https://example.com/hook", cfg.Webhooks[0].URL)
	require.Len(t, cfg.OAuthProviders, 1)
	assert.Equal(t, "google", cfg.OAuthProviders[0].ID)
	assert.Equal(t, "Google", cfg.OAuthProviders[0].Name)
	assert.Equal(t, "https://example.com/auth", cfg.OAuthProviders[0].AuthURL)
}
