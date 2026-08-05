package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const fileName = "easypour.db"

// Path returns the unified SQLite path: same dir as config if set, else ./easypour.db.
func Path(configPath string) string {
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), fileName)
	}
	return fileName
}

// DataDir returns the directory that holds the DB, images, and optional menu.yaml seed.
func DataDir(configPath string) string {
	if configPath != "" {
		return filepath.Dir(configPath)
	}
	return "."
}

// Open opens (or creates) the SQLite database at dbPath.
func Open(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	return db, nil
}

// HasMigration reports whether the given sql-migrate id is present in migrations.
func HasMigration(ctx context.Context, db *sql.DB, id string) (bool, error) {
	var n int
	err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migrations WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

// LatestMigration returns the most recently applied sql-migrate id, or "".
func LatestMigration(ctx context.Context, db *sql.DB) (string, error) {
	var id sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT id FROM migrations ORDER BY applied_at DESC, id DESC LIMIT 1`).Scan(&id)
	if err != nil {
		return "", err
	}
	if !id.Valid {
		return "", nil
	}
	return id.String, nil
}

// ApplySchema creates tables and indexes for unit tests (production uses sql-migrate).
func ApplySchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS menu_items (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			supports_sugar INTEGER NOT NULL,
			supports_milk INTEGER NOT NULL,
			image_url TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS orders (
			id TEXT PRIMARY KEY,
			menu_item_id TEXT NOT NULL,
			username TEXT NOT NULL,
			add_sugar INTEGER NOT NULL,
			add_milk INTEGER NOT NULL,
			sugar_amount INTEGER NOT NULL,
			milk_amount INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			group_id TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_orders_username ON orders(username);
		CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
		CREATE INDEX IF NOT EXISTS idx_orders_group_id ON orders(group_id);
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS cvars (
			cvar_key TEXT NOT NULL PRIMARY KEY,
			cvar_value_int INTEGER NULL,
			cvar_value_string TEXT NULL,
			cvar_main_type TEXT NOT NULL,
			cvar_title TEXT NOT NULL DEFAULT '',
			cvar_description TEXT NOT NULL DEFAULT '',
			cvar_category TEXT NOT NULL DEFAULT '',
			cvar_ordinal INTEGER NOT NULL DEFAULT 0
		);
	`)
	return err
}

// MigrateLegacy copies data from older per-domain DB files into the unified DB when present.
// Existing rows in the destination are left unchanged (INSERT OR IGNORE).
func MigrateLegacy(db *sql.DB, dataDir string) error {
	if err := migrateMenu(db, filepath.Join(dataDir, "menu.db")); err != nil {
		return err
	}
	if err := migrateOrders(db, filepath.Join(dataDir, "orders.db")); err != nil {
		return err
	}
	return migrateSettings(db, filepath.Join(dataDir, "settings.db"))
}

func migrateMenu(db *sql.DB, path string) error {
	return withAttached(db, path, "legacy_menu", func(db *sql.DB) error {
		_, err := db.Exec(`INSERT OR IGNORE INTO menu_items SELECT * FROM legacy_menu.menu_items`)
		return err
	})
}

func migrateOrders(db *sql.DB, path string) error {
	return withAttached(db, path, "legacy_orders", copyOrdersFromLegacy)
}

func copyOrdersFromLegacy(db *sql.DB) error {
	hasGroup, err := columnExists(db, "legacy_orders", "orders", "group_id")
	if err != nil {
		return err
	}
	if hasGroup {
		_, err = db.Exec(`INSERT OR IGNORE INTO orders SELECT * FROM legacy_orders.orders`)
		return err
	}
	_, err = db.Exec(`
		INSERT OR IGNORE INTO orders (
			id, menu_item_id, username, add_sugar, add_milk, sugar_amount, milk_amount,
			status, created_at, updated_at, group_id
		)
		SELECT id, menu_item_id, username, add_sugar, add_milk, sugar_amount, milk_amount,
			status, created_at, updated_at, id
		FROM legacy_orders.orders
	`)
	return err
}

func migrateSettings(db *sql.DB, path string) error {
	return withAttached(db, path, "legacy_settings", copySettingsFromLegacy)
}

func copySettingsFromLegacy(db *sql.DB) error {
	if err := copyTableIfPresent(db, "legacy_settings", "settings",
		`INSERT OR IGNORE INTO settings SELECT * FROM legacy_settings.settings`); err != nil {
		return err
	}
	return copyTableIfPresent(db, "legacy_settings", "cvars",
		`INSERT OR IGNORE INTO cvars SELECT * FROM legacy_settings.cvars`)
}

func copyTableIfPresent(db *sql.DB, schema, table, copySQL string) error {
	ok, err := tableExists(db, schema, table)
	if err != nil || !ok {
		return err
	}
	_, err = db.Exec(copySQL)
	return err
}

func withAttached(db *sql.DB, legacyPath, schema string, fn func(*sql.DB) error) error {
	if !fileExists(legacyPath) {
		return nil
	}
	if _, err := db.Exec(fmt.Sprintf(`ATTACH DATABASE %q AS %s`, legacyPath, schema)); err != nil {
		return fmt.Errorf("attach %s: %w", legacyPath, err)
	}
	fnErr := fn(db)
	_, detachErr := db.Exec(fmt.Sprintf(`DETACH DATABASE %s`, schema))
	return firstErr(wrapMigrate(legacyPath, fnErr), wrapDetach(schema, detachErr))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func wrapMigrate(path string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("migrate from %s: %w", path, err)
}

func wrapDetach(schema string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("detach %s: %w", schema, err)
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func tableExists(db *sql.DB, schema, table string) (bool, error) {
	var name string
	err := db.QueryRow(
		fmt.Sprintf(`SELECT name FROM %s.sqlite_master WHERE type = 'table' AND name = ?`, schema),
		table,
	).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func columnExists(db *sql.DB, schema, table, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf(`PRAGMA %s.table_info(%s)`, schema, table))
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	return scanColumnNames(rows, column)
}

func scanColumnNames(rows *sql.Rows, column string) (bool, error) {
	for rows.Next() {
		name, err := scanPragmaColumnName(rows)
		if err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func scanPragmaColumnName(rows *sql.Rows) (string, error) {
	var cid int
	var name, ctype string
	var notnull, pk int
	var dflt sql.NullString
	err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
	return name, err
}
