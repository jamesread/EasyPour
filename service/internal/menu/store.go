package menu

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	easypourv1 "easypour/service/gen/easypour/v1"
)

// menuYAML is the on-disk format for menu.yaml
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

// Store loads and saves the menu from a YAML file.
type Store struct {
	path string
}

// NewStore returns a store that reads/writes the given path (e.g. menu.yaml).
func NewStore(path string) *Store {
	return &Store{path: path}
}

// GetMenuPath returns a path for menu.yaml next to the config file, or ./menu.yaml.
func GetMenuPath(configPath string) string {
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "menu.yaml")
	}
	return "menu.yaml"
}

// Load reads the menu from disk. If the file is missing, returns default items and does not write.
func (s *Store) Load() ([]*easypourv1.MenuItem, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultItems(), nil
		}
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
	if len(out) == 0 {
		return defaultItems(), nil
	}
	return out, nil
}

// Save writes the given items to disk.
func (s *Store) Save(items []*easypourv1.MenuItem) error {
	m := menuYAML{
		Items: make([]menuItemYAML, 0, len(items)),
	}
	for _, it := range items {
		m.Items = append(m.Items, menuItemYAML{
			ID:            it.Id,
			Name:          it.Name,
			Description:   it.Description,
			SupportsSugar: it.SupportsSugar,
			SupportsMilk:  it.SupportsMilk,
			ImageURL:      it.ImageUrl,
			Category:      it.Category,
		})
	}
	data, err := yaml.Marshal(&m)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Create appends an item (assigning id if empty), saves, and returns the created item.
func (s *Store) Create(item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	items, err := s.Load()
	if err != nil {
		return nil, err
	}
	created := cloneItem(item)
	if created.Id == "" {
		created.Id = "item-" + uuid.New().String()[:8]
	}
	items = append(items, created)
	if err := s.Save(items); err != nil {
		return nil, err
	}
	return created, nil
}

// Update replaces the item with the same id and saves.
func (s *Store) Update(item *easypourv1.MenuItem) (*easypourv1.MenuItem, error) {
	if item.Id == "" {
		return nil, fmt.Errorf("item id required for update")
	}
	items, err := s.Load()
	if err != nil {
		return nil, err
	}
	for i, it := range items {
		if it.Id == item.Id {
			items[i] = cloneItem(item)
			return items[i], s.Save(items)
		}
	}
	return nil, fmt.Errorf("menu item not found: %s", item.Id)
}

// Delete removes the item with the given id and saves.
func (s *Store) Delete(id string) error {
	items, err := s.Load()
	if err != nil {
		return err
	}
	for i, it := range items {
		if it.Id == id {
			items = append(items[:i], items[i+1:]...)
			return s.Save(items)
		}
	}
	return fmt.Errorf("menu item not found: %s", id)
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
