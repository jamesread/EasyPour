package order

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"easypour/service/internal/sqlite"
)

func TestCreateAndListByGroupID(t *testing.T) {
	store := openTestStore(t)

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
	store := openTestStore(t)

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
	store := openTestStore(t)

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

func TestListIncludesGroupID(t *testing.T) {
	store := openTestStore(t)

	require.NoError(t, store.Create(&Order{
		ID: "x", MenuItemID: "latte", Username: "alice", Status: "pending", GroupID: "g1",
	}))

	list, err := store.List("alice", false)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "g1", list[0].GroupID)
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, sqlite.ApplySchema(db))
	return NewStore(db)
}
