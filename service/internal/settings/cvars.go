package settings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CvarRow is one configuration variable row.
type CvarRow struct {
	Key, MainType, Title, Description, Category, ValueString string
	Ordinal, ValueInt                                         int
}

func cvarSelectSQL() string {
	return `SELECT cvar_key, COALESCE(cvar_value_int, 0), COALESCE(cvar_value_string, ''), cvar_main_type,
		COALESCE(cvar_title, ''), COALESCE(cvar_description, ''), COALESCE(cvar_category, ''), COALESCE(cvar_ordinal, 0)
		FROM cvars`
}

func scanCvar(s interface{ Scan(...any) error }) (*CvarRow, error) {
	var row CvarRow
	if err := s.Scan(&row.Key, &row.ValueInt, &row.ValueString, &row.MainType, &row.Title, &row.Description,
		&row.Category, &row.Ordinal); err != nil {
		return nil, err
	}
	return &row, nil
}

// ListCvars returns all cvars ordered by ordinal then key.
func (s *Store) ListCvars(ctx context.Context) ([]CvarRow, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	rows, err := s.db.QueryContext(ctx, cvarSelectSQL()+" ORDER BY cvar_ordinal, cvar_key")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	return collectCvars(rows)
}

func collectCvars(rows *sql.Rows) ([]CvarRow, error) {
	var out []CvarRow
	for rows.Next() {
		row, err := scanCvar(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *row)
	}
	return out, rows.Err()
}

// FindCvar returns a cvar by key, or nil if missing.
func (s *Store) FindCvar(ctx context.Context, key string) (*CvarRow, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.db.QueryRowContext(ctx, cvarSelectSQL()+" WHERE cvar_key = ?", key)
	c, err := scanCvar(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// InsertCvarIfMissing inserts a cvar, or refreshes metadata only on conflict.
func (s *Store) InsertCvarIfMissing(ctx context.Context, row CvarRow) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO cvars (
			cvar_key, cvar_value_int, cvar_value_string, cvar_main_type,
			cvar_title, cvar_description, cvar_category, cvar_ordinal
		) VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, ?, ?)
		ON CONFLICT(cvar_key) DO UPDATE SET
			cvar_title = excluded.cvar_title,
			cvar_description = excluded.cvar_description,
			cvar_category = excluded.cvar_category,
			cvar_ordinal = excluded.cvar_ordinal
	`, row.Key, row.ValueInt, row.ValueString, row.MainType, row.Title, row.Description, row.Category, row.Ordinal)
	if err != nil {
		return fmt.Errorf("insert cvar %s: %w", row.Key, err)
	}
	return nil
}

// UpdateCvar updates only the value columns for a key.
func (s *Store) UpdateCvar(ctx context.Context, key string, valueInt int, valueString string) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE cvars SET cvar_value_int = ?, cvar_value_string = NULLIF(?, '') WHERE cvar_key = ?
	`, valueInt, valueString, key)
	if err != nil {
		return fmt.Errorf("update cvar %s: %w", key, err)
	}
	return ensureRowsAffected(res, key)
}

func ensureRowsAffected(res sql.Result, key string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("cvar not found: %s", key)
	}
	return nil
}
