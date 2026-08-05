package menu

import (
	"database/sql"
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
func (s *Store) SeedIfEmpty() error {
	n, err := s.countItems()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	yamlPath := filepath.Join(s.dataDir, "menu.yaml")
	if items, err := loadYAMLFile(yamlPath); err == nil && len(items) > 0 {
		return s.insertAll(items)
	}
	return s.insertAll(defaultItems())
}

func (s *Store) countItems() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM menu_items`).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count menu_items: %w", err)
	}
	return n, nil
}

func (s *Store) insertAll(items []*easypourv1.MenuItem) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(`
		INSERT INTO menu_items (id, name, description, supports_sugar, supports_milk, image_url, category)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare seed insert: %w", err)
	}
	defer stmt.Close()

	for _, it := range items {
		if _, err := stmt.Exec(
			it.Id, it.Name, it.Description,
			boolToInt(it.SupportsSugar), boolToInt(it.SupportsMilk),
			it.ImageUrl, it.Category,
		); err != nil {
			return fmt.Errorf("seed insert %s: %w", it.Id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit seed: %w", err)
	}
	return nil
}

// Load returns all menu items.
func (s *Store) Load() ([]*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	rows, err := s.db.Query(`SELECT ` + menuItemSelectCols + ` FROM menu_items ORDER BY category, name`)
	if err != nil {
		return nil, fmt.Errorf("list menu_items: %w", err)
	}
	defer rows.Close()

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
func (s *Store) Create(item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if item == nil {
		return nil, fmt.Errorf("item required")
	}
	created := cloneItem(item)
	if created.Id == "" {
		created.Id = "item-" + uuid.New().String()[:8]
	}
	_, err := s.db.Exec(`
		INSERT INTO menu_items (id, name, description, supports_sugar, supports_milk, image_url, category)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, created.Id, created.Name, created.Description,
		boolToInt(created.SupportsSugar), boolToInt(created.SupportsMilk),
		created.ImageUrl, created.Category)
	if err != nil {
		return nil, fmt.Errorf("create menu item: %w", err)
	}
	return created, nil
}

// Update replaces the item with the same id.
func (s *Store) Update(item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if item == nil || item.Id == "" {
		return nil, fmt.Errorf("item id required for update")
	}
	updated := cloneItem(item)
	res, err := s.db.Exec(`
		UPDATE menu_items
		SET name = ?, description = ?, supports_sugar = ?, supports_milk = ?, image_url = ?, category = ?
		WHERE id = ?
	`, updated.Name, updated.Description,
		boolToInt(updated.SupportsSugar), boolToInt(updated.SupportsMilk),
		updated.ImageUrl, updated.Category, updated.Id)
	if err != nil {
		return nil, fmt.Errorf("update menu item: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("update menu item: %w", err)
	}
	if n == 0 {
		return nil, fmt.Errorf("menu item not found: %s", item.Id)
	}
	return updated, nil
}

// Delete removes the item with the given id.
func (s *Store) Delete(id string) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	if id == "" {
		return fmt.Errorf("item id required for delete")
	}
	res, err := s.db.Exec(`DELETE FROM menu_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete menu item: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete menu item: %w", err)
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
	data, err := os.ReadFile(path)
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
