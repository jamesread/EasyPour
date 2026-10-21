package menu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	easypourv1 "easypour/service/gen/easypour/v1"
)

// menuYAML is the on-disk format for optional menu.yaml seed import.
type menuYAML struct {
	Items []menuItemYAML `yaml:"items"`
}

type menuItemYAML struct {
	ID            string `yaml:"id"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	SupportsSugar bool   `yaml:"supports_sugar"`
	SupportsMilk  bool   `yaml:"supports_milk"`
	ImageURL      string `yaml:"image_url"`
	Category      string `yaml:"category"`
}

const menuItemSelectCols = `id, name, description, supports_sugar, supports_milk, image_url, category`

const insertMenuItemSQL = `
	INSERT INTO menu_items (id, name, description, supports_sugar, supports_milk, image_url, category)
	VALUES (?, ?, ?, ?, ?, ?, ?)
`

var errNotInitialized = errors.New("store not initialized")

// Store persists menu items in SQLite.
type Store struct {
	db      *sql.DB
	dataDir string
}

// GetMenuPath returns a path for legacy menu.yaml next to the config file, or ./menu.yaml.
// Used only as a one-shot import seed when the database is empty.
func GetMenuPath(configPath string) string {
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "menu.yaml")
	}
	return "menu.yaml"
}

// NewStore wraps a shared SQLite database. dataDir holds optional menu.yaml for first-boot seed.
func NewStore(db *sql.DB, dataDir string) *Store {
	if dataDir == "" {
		dataDir = "."
	}
	return &Store{db: db, dataDir: dataDir}
}

// SeedIfEmpty imports menu.yaml or built-in defaults when the menu table has no rows.
func (s *Store) SeedIfEmpty(ctx context.Context) error {
	n, err := s.countItems(ctx)
	if err != nil || n > 0 {
		return err
	}
	return s.insertAll(ctx, s.seedItems())
}

func (s *Store) seedItems() []*easypourv1.MenuItem {
	items, err := loadYAMLFile(filepath.Join(s.dataDir, "menu.yaml"))
	if err == nil && len(items) > 0 {
		return items
	}
	return defaultItems()
}

func (s *Store) countItems(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM menu_items`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count menu_items: %w", err)
	}
	return n, nil
}

func (s *Store) insertAll(ctx context.Context, items []*easypourv1.MenuItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := insertItems(ctx, tx, items); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

func insertItems(ctx context.Context, tx *sql.Tx, items []*easypourv1.MenuItem) error {
	stmt, err := tx.PrepareContext(ctx, insertMenuItemSQL)
	if err != nil {
		return fmt.Errorf("prepare seed insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, it := range items {
		if _, err := stmt.ExecContext(ctx, insertArgs(it)...); err != nil {
			return fmt.Errorf("seed insert %s: %w", it.Id, err)
		}
	}
	return nil
}

func insertArgs(it *easypourv1.MenuItem) []any {
	return []any{
		it.Id, it.Name, it.Description,
		boolToInt(it.SupportsSugar), boolToInt(it.SupportsMilk),
		it.ImageUrl, it.Category,
	}
}

// Load returns all menu items.
func (s *Store) Load(ctx context.Context) ([]*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, errNotInitialized
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+menuItemSelectCols+` FROM menu_items ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list menu_items: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanMenuItems(rows)
}

func scanMenuItems(rows *sql.Rows) ([]*easypourv1.MenuItem, error) {
	var out []*easypourv1.MenuItem
	for rows.Next() {
		it, err := scanMenuItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list menu_items: %w", err)
	}
	return out, nil
}

// Create inserts an item (assigning id if empty) and returns the created item.
func (s *Store) Create(ctx context.Context, item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, errNotInitialized
	}
	if item == nil {
		return nil, fmt.Errorf("item required")
	}
	created := withGeneratedID(cloneItem(item))
	if _, err := s.db.ExecContext(ctx, insertMenuItemSQL, insertArgs(created)...); err != nil {
		return nil, fmt.Errorf("create menu item: %w", err)
	}
	return created, nil
}

func withGeneratedID(it *easypourv1.MenuItem) *easypourv1.MenuItem {
	if it.Id == "" {
		it.Id = "item-" + uuid.New().String()[:8]
	}
	return it
}

// Update replaces the item with the same id.
func (s *Store) Update(ctx context.Context, item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, errNotInitialized
	}
	if item.GetId() == "" {
		return nil, fmt.Errorf("item id required for update")
	}
	updated := cloneItem(item)
	err := s.execAffectingItem(ctx, "update menu item", updated.Id, `
		UPDATE menu_items
		SET name = ?, description = ?, supports_sugar = ?, supports_milk = ?, image_url = ?, category = ?
		WHERE id = ?
	`, updated.Name, updated.Description,
		boolToInt(updated.SupportsSugar), boolToInt(updated.SupportsMilk),
		updated.ImageUrl, updated.Category, updated.Id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// Delete removes the item with the given id.
func (s *Store) Delete(ctx context.Context, id string) error {
	if s.db == nil {
		return errNotInitialized
	}
	if id == "" {
		return fmt.Errorf("item id required for delete")
	}
	return s.execAffectingItem(ctx, "delete menu item", id, `DELETE FROM menu_items WHERE id = ?`, id)
}

func (s *Store) execAffectingItem(ctx context.Context, op, id, query string, args ...any) error {
	res, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	return requireItemAffected(res, op, id)
}

func requireItemAffected(res sql.Result, op, id string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	if n == 0 {
		return fmt.Errorf("menu item not found: %s", id)
	}
	return nil
}

func scanMenuItem(rows *sql.Rows) (*easypourv1.MenuItem, error) {
	var (
		it            easypourv1.MenuItem
		supportsSugar int
		supportsMilk  int
	)
	err := rows.Scan(
		&it.Id, &it.Name, &it.Description,
		&supportsSugar, &supportsMilk,
		&it.ImageUrl, &it.Category,
	)
	if err != nil {
		return nil, fmt.Errorf("scan menu item: %w", err)
	}
	it.SupportsSugar = supportsSugar != 0
	it.SupportsMilk = supportsMilk != 0
	return &it, nil
}

func loadYAMLFile(path string) ([]*easypourv1.MenuItem, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var m menuYAML
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("menu yaml: %w", err)
	}
	out := make([]*easypourv1.MenuItem, 0, len(m.Items))
	for _, it := range m.Items {
		out = append(out, &easypourv1.MenuItem{
			Id:            it.ID,
			Name:          it.Name,
			Description:   it.Description,
			SupportsSugar: it.SupportsSugar,
			SupportsMilk:  it.SupportsMilk,
			ImageUrl:      it.ImageURL,
			Category:      it.Category,
		})
	}
	return out, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func defaultItems() []*easypourv1.MenuItem {
	return []*easypourv1.MenuItem{
		{Id: "coffee", Name: "Coffee", Description: "Freshly brewed coffee", SupportsSugar: true, SupportsMilk: true, Category: "Drinks"},
		{Id: "tea", Name: "Tea", Description: "Hot tea", SupportsSugar: true, SupportsMilk: true, Category: "Drinks"},
		{Id: "hot-chocolate", Name: "Hot Chocolate", Description: "Rich hot chocolate", SupportsSugar: false, SupportsMilk: true, Category: "Drinks"},
	}
}

func cloneItem(it *easypourv1.MenuItem) *easypourv1.MenuItem {
	return &easypourv1.MenuItem{
		Id:            it.Id,
		Name:          it.Name,
		Description:   it.Description,
		SupportsSugar: it.SupportsSugar,
		SupportsMilk:  it.SupportsMilk,
		ImageUrl:      it.ImageUrl,
		Category:      it.Category,
	}
}
