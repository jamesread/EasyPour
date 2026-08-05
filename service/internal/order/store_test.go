package order

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateAndListByGroupID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())

	groupID := "group-1"
	require.NoError(t, store.Create(&Order{
		ID: "order-a", MenuItemID: "latte", Username: "alice",
		Status: "pending", GroupID: groupID,
	}))
	require.NoError(t, store.Create(&Order{
		ID: "order-b", MenuItemID: "tea", Username: "alice",
		Status: "pending", GroupID: groupID,
	}))
	require.NoError(t, store.Create(&Order{
		ID: "order-c", MenuItemID: "mocha", Username: "bob",
		Status: "pending", GroupID: "group-2",
	}))

	got, err := store.ListByGroupID(groupID)
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := []string{got[0].ID, got[1].ID}
	require.ElementsMatch(t, []string{"order-a", "order-b"}, ids)
}

func TestCreateDefaultsGroupIDToOrderID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())

	require.NoError(t, store.Create(&Order{
		ID: "solo-1", MenuItemID: "water", Username: "alice", Status: "pending",
	}))

	got, err := store.Get("solo-1")
	require.NoError(t, err)
	require.Equal(t, "solo-1", got.GroupID)

	list, err := store.ListByGroupID("solo-1")
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestUpdateStatusByGroupID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())

	groupID := "group-upd"
	require.NoError(t, store.Create(&Order{
		ID: "a", MenuItemID: "latte", Username: "alice", Status: "pending", GroupID: groupID,
	}))
	require.NoError(t, store.Create(&Order{
		ID: "b", MenuItemID: "tea", Username: "alice", Status: "pending", GroupID: groupID,
	}))

	updated, err := store.UpdateStatusByGroupID(groupID, "delivered")
	require.NoError(t, err)
	require.Len(t, updated, 2)
	for _, o := range updated {
		require.Equal(t, "delivered", o.Status)
	}
}

func TestInitMigratesLegacyOrdersTableWithoutGroupID(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "orders.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`
		CREATE TABLE orders (
			id TEXT PRIMARY KEY,
			menu_item_id TEXT NOT NULL,
			username TEXT NOT NULL,
			add_sugar INTEGER NOT NULL,
			add_milk INTEGER NOT NULL,
			sugar_amount INTEGER NOT NULL,
			milk_amount INTEGER NOT NULL,
			status TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
		INSERT INTO orders VALUES ('legacy-1', 'latte', 'alice', 0, 0, 0, 0, 'pending', 1, 1);
	`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	store, err := NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())

	got, err := store.Get("legacy-1")
	require.NoError(t, err)
	require.Equal(t, "legacy-1", got.GroupID)
}

func TestListIncludesGroupID(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(filepath.Join(dir, "orders.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	require.NoError(t, store.Init())

	require.NoError(t, store.Create(&Order{
		ID: "x", MenuItemID: "latte", Username: "alice", Status: "pending", GroupID: "g1",
	}))

	list, err := store.List("alice", false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "g1", list[0].GroupID)
}
