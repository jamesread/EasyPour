package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPath_UsesConfigDir(t *testing.T) {
	assert.Equal(t, filepath.Join("/data", "easypour.db"), Path("/data/config.yaml"))
}

func TestPath_DefaultWhenEmpty(t *testing.T) {
	assert.Equal(t, "easypour.db", Path(""))
}

func TestDataDir(t *testing.T) {
	assert.Equal(t, "/data", DataDir("/data/config.yaml"))
	assert.Equal(t, ".", DataDir(""))
}

func TestHasMigration_AndLatest(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE migrations (
			id TEXT PRIMARY KEY,
			applied_at DATETIME
		);
		INSERT INTO migrations (id, applied_at) VALUES ('0.base.sql', datetime('now'));
	`)
	require.NoError(t, err)

	ctx := context.Background()
	ok, err := HasMigration(ctx, db, "0.base.sql")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = HasMigration(ctx, db, "missing.sql")
	require.NoError(t, err)
	assert.False(t, ok)

	latest, err := LatestMigration(ctx, db)
	require.NoError(t, err)
	assert.Equal(t, "0.base.sql", latest)
}

func TestOpen_AndMigrateLegacy(t *testing.T) {
	dir := t.TempDir()
	mustCreateLegacyMenu(t, dir)
	mustCreateLegacyOrders(t, dir)
	mustCreateLegacySettings(t, dir)

	dbPath := filepath.Join(dir, "easypour.db")
	db, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, ApplySchema(db))
	require.NoError(t, MigrateLegacy(db, dir))

	var menuCount, orderCount, cvarCount int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM menu_items`).Scan(&menuCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&orderCount))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM cvars`).Scan(&cvarCount))
	assert.Equal(t, 1, menuCount)
	assert.Equal(t, 1, orderCount)
	assert.Equal(t, 1, cvarCount)

	require.NoError(t, MigrateLegacy(db, dir))
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM menu_items`).Scan(&menuCount))
	assert.Equal(t, 1, menuCount)
}

func TestMigrateLegacy_OrdersWithoutGroupID(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY, menu_item_id TEXT NOT NULL, username TEXT NOT NULL,
			add_sugar INTEGER NOT NULL, add_milk INTEGER NOT NULL,
			sugar_amount INTEGER NOT NULL, milk_amount INTEGER NOT NULL,
			status TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL
		);
		INSERT INTO orders VALUES ('o1', 'm1', 'u', 0, 0, 0, 0, 'pending', 1, 1);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	unified, err := Open(filepath.Join(dir, "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = unified.Close() })
	require.NoError(t, ApplySchema(unified))
	require.NoError(t, MigrateLegacy(unified, dir))

	var groupID string
	require.NoError(t, unified.QueryRow(`SELECT group_id FROM orders WHERE id = 'o1'`).Scan(&groupID))
	assert.Equal(t, "o1", groupID)
}

func mustCreateLegacyMenu(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "menu.db"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE menu_items (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, description TEXT NOT NULL,
			supports_sugar INTEGER NOT NULL, supports_milk INTEGER NOT NULL,
			image_url TEXT NOT NULL DEFAULT '', category TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO menu_items VALUES ('m1', 'Tea', '', 0, 0, '', 'Drinks');
	`)
	require.NoError(t, err)
}

func mustCreateLegacyOrders(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY, menu_item_id TEXT NOT NULL, username TEXT NOT NULL,
			add_sugar INTEGER NOT NULL, add_milk INTEGER NOT NULL,
			sugar_amount INTEGER NOT NULL, milk_amount INTEGER NOT NULL,
			status TEXT NOT NULL, created_at INTEGER NOT NULL, updated_at INTEGER NOT NULL,
			group_id TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO orders VALUES ('o1', 'm1', 'u', 0, 0, 0, 0, 'pending', 1, 1, 'o1');
	`)
	require.NoError(t, err)
}

func mustCreateLegacySettings(t *testing.T, dir string) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dir, "settings.db"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`
		CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		CREATE TABLE cvars (
			cvar_key TEXT NOT NULL PRIMARY KEY,
			cvar_value_int INTEGER NULL,
			cvar_value_string TEXT NULL,
			cvar_main_type TEXT NOT NULL,
			cvar_title TEXT NOT NULL DEFAULT '',
			cvar_description TEXT NOT NULL DEFAULT '',
			cvar_category TEXT NOT NULL DEFAULT '',
			cvar_ordinal INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO cvars (cvar_key, cvar_value_string, cvar_main_type, cvar_title, cvar_ordinal)
		VALUES ('site_title', 'Legacy', 'string', 'Site title', 10);
	`)
	require.NoError(t, err)
}
