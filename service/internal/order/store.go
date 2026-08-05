package order

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	easypourv1 "easypour/service/gen/easypour/v1"
)

// Order is the in-memory representation of a persisted order (matches DB schema).
type Order struct {
	ID          string
	MenuItemID  string
	Username    string
	AddSugar    bool
	AddMilk     bool
	SugarAmount int32
	MilkAmount  int32
	Status      string
	CreatedAt   int64
	UpdatedAt   int64
	GroupID     string
}

// Store persists orders in SQLite.
type Store struct {
	db *sql.DB
}

// GetOrdersDBPath returns the path for orders.db: same dir as config if config path set, else ./orders.db.
func GetOrdersDBPath(configPath string) string {
	if configPath != "" {
		return filepath.Join(filepath.Dir(configPath), "orders.db")
	}
	return "orders.db"
}

// NewStore opens the SQLite database at dbPath and returns a store. Caller should call Init() then use the store; defer store.Close() when done.
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

// Init creates the orders table and indexes if they do not exist.
func (s *Store) Init() error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	_, err := s.db.Exec(`
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
	`)
	if err != nil {
		return fmt.Errorf("init orders table: %w", err)
	}
	if err := s.ensureGroupIDColumn(); err != nil {
		return err
	}
	return nil
}

func (s *Store) ensureGroupIDColumn() error {
	_, err := s.db.Exec(`ALTER TABLE orders ADD COLUMN group_id TEXT NOT NULL DEFAULT ''`)
	if err != nil {
		// Column already exists on freshly created tables / prior migrations.
		_ = err
	}
	_, err = s.db.Exec(`UPDATE orders SET group_id = id WHERE group_id = '' OR group_id IS NULL`)
	if err != nil {
		return fmt.Errorf("backfill group_id: %w", err)
	}
	_, err = s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_group_id ON orders(group_id)`)
	if err != nil {
		return fmt.Errorf("index group_id: %w", err)
	}
	return nil
}

// Create persists an order. Order.ID must be set; CreatedAt/UpdatedAt set to now if zero.
// Empty GroupID defaults to the order ID.
func (s *Store) Create(o *Order) error {
	if s.db == nil {
		return fmt.Errorf("store not initialized")
	}
	if o.ID == "" {
		return fmt.Errorf("order id required")
	}
	if o.GroupID == "" {
		o.GroupID = o.ID
	}
	now := time.Now().Unix()
	if o.CreatedAt == 0 {
		o.CreatedAt = now
	}
	if o.UpdatedAt == 0 {
		o.UpdatedAt = now
	}
	_, err := s.db.Exec(`
		INSERT INTO orders (id, menu_item_id, username, add_sugar, add_milk, sugar_amount, milk_amount, status, created_at, updated_at, group_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.MenuItemID, o.Username, boolToInt(o.AddSugar), boolToInt(o.AddMilk), o.SugarAmount, o.MilkAmount, o.Status, o.CreatedAt, o.UpdatedAt, o.GroupID)
	if err != nil {
		return fmt.Errorf("create order: %w", err)
	}
	return nil
}

func scanOrder(scanner interface {
	Scan(dest ...any) error
}) (*Order, error) {
	var o Order
	var addSugar, addMilk int
	err := scanner.Scan(
		&o.ID, &o.MenuItemID, &o.Username, &addSugar, &addMilk,
		&o.SugarAmount, &o.MilkAmount, &o.Status, &o.CreatedAt, &o.UpdatedAt, &o.GroupID,
	)
	if err != nil {
		return nil, err
	}
	o.AddSugar = addSugar != 0
	o.AddMilk = addMilk != 0
	if o.GroupID == "" {
		o.GroupID = o.ID
	}
	return &o, nil
}

const orderSelectCols = `id, menu_item_id, username, add_sugar, add_milk, sugar_amount, milk_amount, status, created_at, updated_at, group_id`

// Get returns the order by id, or nil if not found.
func (s *Store) Get(id string) (*Order, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	row := s.db.QueryRow(`SELECT `+orderSelectCols+` FROM orders WHERE id = ?`, id)
	o, err := scanOrder(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}
	return o, nil
}

// ListByGroupID returns all orders sharing a group id, oldest first.
func (s *Store) ListByGroupID(groupID string) ([]*Order, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if groupID == "" {
		return nil, nil
	}
	rows, err := s.db.Query(`
		SELECT `+orderSelectCols+`
		FROM orders WHERE group_id = ? ORDER BY created_at ASC
	`, groupID)
	if err != nil {
		return nil, fmt.Errorf("list by group: %w", err)
	}
	defer rows.Close()
	return scanOrders(rows)
}

// List returns orders: when isAdmin is true, all orders; otherwise only orders for the given username. Newest first.
func (s *Store) List(username string, isAdmin bool) ([]*Order, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	var rows *sql.Rows
	var err error
	if isAdmin {
		rows, err = s.db.Query(`
			SELECT ` + orderSelectCols + `
			FROM orders ORDER BY created_at DESC
		`)
	} else {
		rows, err = s.db.Query(`
			SELECT `+orderSelectCols+`
			FROM orders WHERE username = ? ORDER BY created_at DESC
		`, username)
	}
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	return scanOrders(rows)
}

func scanOrders(rows *sql.Rows) ([]*Order, error) {
	var list []*Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		list = append(list, o)
	}
	return list, rows.Err()
}

// UpdateStatus sets the order status and updated_at. Returns the updated order or nil if not found.
func (s *Store) UpdateStatus(id, status string) (*Order, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(`UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`, status, now, id)
	if err != nil {
		return nil, fmt.Errorf("update order status: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	return s.Get(id)
}

// UpdateStatusByGroupID sets status for every order in the group. Returns the updated orders.
func (s *Store) UpdateStatusByGroupID(groupID, status string) ([]*Order, error) {
	if s.db == nil {
		return nil, fmt.Errorf("store not initialized")
	}
	if groupID == "" {
		return nil, nil
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`UPDATE orders SET status = ?, updated_at = ? WHERE group_id = ?`, status, now, groupID)
	if err != nil {
		return nil, fmt.Errorf("update group status: %w", err)
	}
	return s.ListByGroupID(groupID)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// ToProto returns the protobuf Order message for this order.
func (o *Order) ToProto() *easypourv1.Order {
	if o == nil {
		return nil
	}
	return &easypourv1.Order{
		OrderId:     o.ID,
		MenuItemId:  o.MenuItemID,
		Username:    o.Username,
		AddSugar:    o.AddSugar,
		AddMilk:     o.AddMilk,
		SugarAmount: o.SugarAmount,
		MilkAmount:  o.MilkAmount,
		Status:      o.Status,
		CreatedAt:   o.CreatedAt,
		UpdatedAt:   o.UpdatedAt,
		GroupId:     o.GroupID,
	}
}

// OrderFromProto builds an Order from a protobuf Order (e.g. for Create). CreatedAt/UpdatedAt can be zero.
func OrderFromProto(p *easypourv1.Order) *Order {
	if p == nil {
		return nil
	}
	return &Order{
		ID:          p.GetOrderId(),
		MenuItemID:  p.GetMenuItemId(),
		Username:    p.GetUsername(),
		AddSugar:    p.GetAddSugar(),
		AddMilk:     p.GetAddMilk(),
		SugarAmount: p.GetSugarAmount(),
		MilkAmount:  p.GetMilkAmount(),
		Status:      p.GetStatus(),
		CreatedAt:   p.GetCreatedAt(),
		UpdatedAt:   p.GetUpdatedAt(),
		GroupID:     p.GetGroupId(),
	}
}
