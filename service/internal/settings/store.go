package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"easypour/service/internal/cvar"
)

// Settings holds runtime-configurable application settings (legacy DTO for Apprise).
type Settings struct {
	AppriseURL string
}

// Store persists settings and cvars in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore wraps a shared SQLite database.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// EnsureDefaultCvars inserts missing catalog keys and refreshes metadata only on conflict.
func (s *Store) EnsureDefaultCvars(ctx context.Context, siteTitle string) error {
	for _, def := range cvar.Defaults(siteTitle) {
		if err := s.InsertCvarIfMissing(ctx, CvarRow{
			Key: def.Key, MainType: def.MainType,
			Title: def.Title, Description: def.Description,
			Category: def.Category, Ordinal: def.Ordinal,
			ValueInt: def.ValueInt, ValueString: def.ValueString,
		}); err != nil {
			return err
		}
	}
	return s.migrateLegacyAppriseURL(ctx)
}

// Get returns the current settings (Apprise URL from cvars).
func (s *Store) Get() (*Settings, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row, err := s.FindCvar(context.Background(), cvar.KeyAppriseURL)
	if err != nil {
		return nil, err
	}
	out := &Settings{}
	if row != nil {
		out.AppriseURL = row.ValueString
	}
	return out, nil
}

// Update persists settings (Apprise URL into the apprise_url cvar).
func (s *Store) Update(in *Settings) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	if in == nil {
		return fmt.Errorf("settings required")
	}
	url := strings.TrimSpace(in.AppriseURL)
	return s.UpdateCvar(context.Background(), cvar.KeyAppriseURL, 0, url)
}

func (s *Store) migrateLegacyAppriseURL(ctx context.Context) error {
	legacy, err := s.readLegacyAppriseURL(ctx)
	if err != nil || legacy == "" {
		return err
	}
	return s.copyLegacyIntoEmptyCvar(ctx, cvar.KeyAppriseURL, legacy)
}

func (s *Store) copyLegacyIntoEmptyCvar(ctx context.Context, key, value string) error {
	row, err := s.FindCvar(ctx, key)
	if err != nil || shouldSkipLegacyMigrate(row) {
		return err
	}
	return s.UpdateCvar(ctx, key, 0, value)
}

func (s *Store) readLegacyAppriseURL(ctx context.Context) (string, error) {
	var legacy string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, cvar.KeyAppriseURL).Scan(&legacy)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read legacy apprise_url: %w", err)
	}
	return strings.TrimSpace(legacy), nil
}

func shouldSkipLegacyMigrate(row *CvarRow) bool {
	return row == nil || row.ValueString != ""
}
