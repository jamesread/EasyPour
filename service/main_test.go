package main

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"easypour/service/buildinfo"
	easypourv1 "easypour/service/gen/easypour/v1"
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
	assert.Empty(t, resp.Msg.OauthProviders)
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
