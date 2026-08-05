package main

import (
	"context"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easypour/service/buildinfo"
	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/internal/sqlite"
)

func TestInit_ReturnsVersionAndOAuthProviders(t *testing.T) {
	server := &EasyPourServer{
		oauthProviders: []*easypourv1.OAuthProvider{
			{Id: "google", Name: "Google", AuthUrl: "https://example.com/auth/google"},
		},
	}
	ctx := context.Background()
	req := connect.NewRequest(&easypourv1.InitRequest{})

	resp, err := server.Init(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, buildinfo.Version, resp.Msg.Version)
	assert.Equal(t, "EasyPour", resp.Msg.SiteTitle)
	require.NotNil(t, resp.Msg.Features)
	assert.Len(t, resp.Msg.OauthProviders, 1)
	assert.Equal(t, "google", resp.Msg.OauthProviders[0].Id)
	assert.Equal(t, "Google", resp.Msg.OauthProviders[0].Name)
	assert.Equal(t, "https://example.com/auth/google", resp.Msg.OauthProviders[0].AuthUrl)
}

func TestInit_ReturnsVersionWhenNoOAuthProviders(t *testing.T) {
	server := &EasyPourServer{oauthProviders: nil}
	ctx := context.Background()
	req := connect.NewRequest(&easypourv1.InitRequest{})

	resp, err := server.Init(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, buildinfo.Version, resp.Msg.Version)
	assert.Equal(t, "EasyPour", resp.Msg.SiteTitle)
	assert.Empty(t, resp.Msg.OauthProviders)
}

func TestListCvars_RequiresAdmin(t *testing.T) {
	server := &EasyPourServer{}
	_, err := server.ListCvars(context.Background(), connect.NewRequest(&easypourv1.ListCvarsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestUpdateCvar_RequiresAdmin(t *testing.T) {
	server := &EasyPourServer{}
	_, err := server.UpdateCvar(context.Background(), connect.NewRequest(&easypourv1.UpdateCvarRequest{
		Key: "site_title", ValueString: "x",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestGetSettings_RequiresAdmin(t *testing.T) {
	server := &EasyPourServer{}
	_, err := server.GetSettings(context.Background(), connect.NewRequest(&easypourv1.GetSettingsRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestUpdateSettings_RequiresAdmin(t *testing.T) {
	server := &EasyPourServer{}
	_, err := server.UpdateSettings(context.Background(), connect.NewRequest(&easypourv1.UpdateSettingsRequest{
		Settings: &easypourv1.Settings{AppriseUrl: "http://example/notify"},
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestTestAppriseNotification_RequiresAdmin(t *testing.T) {
	server := &EasyPourServer{}
	_, err := server.TestAppriseNotification(context.Background(), connect.NewRequest(&easypourv1.TestAppriseNotificationRequest{
		AppriseUrl: "http://example/notify",
	}))
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))
}

func TestAssertMigration_MissingTable(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	err = assertMigration(context.Background(), db, "0.base.sql")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run sql-migrate up")
}

func TestAssertMigration_WrongVersion(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE migrations (id TEXT PRIMARY KEY, applied_at DATETIME);
		INSERT INTO migrations (id, applied_at) VALUES ('old.sql', datetime('now'));
	`)
	require.NoError(t, err)

	err = assertMigration(context.Background(), db, "0.base.sql")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires database version 0.base.sql")
	assert.Contains(t, err.Error(), "old.sql")
}

func TestAssertMigration_OK(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "easypour.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec(`
		CREATE TABLE migrations (id TEXT PRIMARY KEY, applied_at DATETIME);
		INSERT INTO migrations (id, applied_at) VALUES ('0.base.sql', datetime('now'));
	`)
	require.NoError(t, err)

	require.NoError(t, assertMigration(context.Background(), db, "0.base.sql"))
}

