package settings

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const keyAppriseURL = "apprise_url"

// Settings holds runtime-configurable application settings.
type Settings struct {
	AppriseURL string
}

// Store persists settings in SQLite.
type Store struct {
	db *sql.DB
}

// GetSettingsDBPath returns the path for settings.db: same dir as config if set, else ./settings.db.
func GetSettingsDBPath(configPath string) string {
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "settings.db")
	}
	return "settings.db"
}

// NewStore opens the SQLite database at dbPath.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Init creates the settings table if it does not exist.
func (s *Store) Init() error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)
	`)
	if err != nil {
		return fmt.Errorf("init settings table: %w", err)
	}
	return nil
}

// Get returns the current settings.
func (s *Store) Get() (*Settings, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	out := &Settings{}
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, keyAppriseURL).Scan(&value)
	if err == sql.ErrNoRows {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get apprise_url: %w", err)
	}
	out.AppriseURL = value
	return out, nil
}

// Update persists settings.
func (s *Store) Update(in *Settings) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	if in == nil {
		return fmt.Errorf("settings required")
	}
	url := strings.TrimSpace(in.AppriseURL)
	_, err := s.db.Exec(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, keyAppriseURL, url)
	if err != nil {
		return fmt.Errorf("update apprise_url: %w", err)
	}
	return nil
}
