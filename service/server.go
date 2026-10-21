package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	auth "github.com/jamesread/httpauthshim"
	"google.golang.org/protobuf/types/known/emptypb"

	"easypour/service/buildinfo"
	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/gen/easypour/v1/easypourv1connect"
	"easypour/service/internal/apprise"
	"easypour/service/internal/config"
	"easypour/service/internal/menu"
	"easypour/service/internal/order"
	"easypour/service/internal/settings"
)

// EasyPourServer implements the EasyPourService
type EasyPourServer struct {
	easypourv1connect.UnimplementedEasyPourServiceHandler
	authCtx        *auth.AuthShimContext
	menuStore      *menu.Store
	orderStore     *order.Store
	settingsStore  *settings.Store
	webhooks       []config.Webhook
	webhookClient  *http.Client                // skips TLS cert verification for webhook POSTs
	oauthProviders []*easypourv1.OAuthProvider // configured OAuth2 providers for login form
	sseBroadcast   *sseBroadcaster
	appriseMu      sync.Mutex
	appriseTimers  map[string]*time.Timer // debounce Apprise notify per order group
}

// Init returns app version, site title, features, and OAuth2 providers. Callable unauthenticated.
func (s *EasyPourServer) Init(
	ctx context.Context,
	_ *connect.Request[easypourv1.InitRequest],
) (*connect.Response[easypourv1.InitResponse], error) {
	return connect.NewResponse(&easypourv1.InitResponse{
		Version:        buildinfo.Version,
		SiteTitle:      s.siteTitle(ctx),
		Features:       s.initFeatures(ctx),
		OauthProviders: s.oauthProviders,
	}), nil
}

// GetMenu returns the available drinks menu.
func (s *EasyPourServer) GetMenu(
	ctx context.Context,
	_ *connect.Request[easypourv1.GetMenuRequest],
) (*connect.Response[easypourv1.GetMenuResponse], error) {
	items, err := s.menuStore.Load(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load menu: %w", err))
	}
	return connect.NewResponse(&easypourv1.GetMenuResponse{Items: items}), nil
}

// menuNameByID returns an empty map when the menu cannot be loaded; callers fall back to item ids.
func (s *EasyPourServer) menuNameByID(ctx context.Context) map[string]string {
	names := map[string]string{}
	for _, it := range s.loadMenuOrEmpty(ctx) {
		if it.GetId() != "" {
			names[it.GetId()] = it.GetName()
		}
	}
	return names
}

func (s *EasyPourServer) loadMenuOrEmpty(ctx context.Context) []*easypourv1.MenuItem {
	if s.menuStore == nil {
		return nil
	}
	items, err := s.menuStore.Load(ctx)
	if err != nil {
		return nil
	}
	return items
}

// CreateMenuItem adds a menu item (admin only)
func (s *EasyPourServer) CreateMenuItem(
	ctx context.Context,
	req *connect.Request[easypourv1.CreateMenuItemRequest],
) (*connect.Response[easypourv1.MenuItem], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	item := req.Msg.GetItem()
	if item == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item required"))
	}
	created, err := s.menuStore.Create(ctx, item)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create menu item: %w", err))
	}
	return connect.NewResponse(created), nil
}

// UpdateMenuItem updates a menu item (admin only)
func (s *EasyPourServer) UpdateMenuItem(
	ctx context.Context,
	req *connect.Request[easypourv1.UpdateMenuItemRequest],
) (*connect.Response[easypourv1.MenuItem], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	item := req.Msg.GetItem()
	if item.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item with id required"))
	}
	updated, err := s.menuStore.Update(ctx, item)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("update menu item: %w", err))
	}
	return connect.NewResponse(updated), nil
}

// DeleteMenuItem removes a menu item by id (admin only)
func (s *EasyPourServer) DeleteMenuItem(
	ctx context.Context,
	req *connect.Request[easypourv1.DeleteMenuItemRequest],
) (*connect.Response[emptypb.Empty], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id required"))
	}
	if err := s.menuStore.Delete(ctx, id); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("delete menu item: %w", err))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetSettings returns runtime settings (admin only).
func (s *EasyPourServer) GetSettings(
	ctx context.Context,
	_ *connect.Request[easypourv1.GetSettingsRequest],
) (*connect.Response[easypourv1.GetSettingsResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	current, err := s.currentSettings(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&easypourv1.GetSettingsResponse{Settings: current}), nil
}

// UpdateSettings updates runtime settings (admin only).
func (s *EasyPourServer) UpdateSettings(
	ctx context.Context,
	req *connect.Request[easypourv1.UpdateSettingsRequest],
) (*connect.Response[easypourv1.UpdateSettingsResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if err := s.saveSettings(ctx, req.Msg.GetSettings()); err != nil {
		return nil, err
	}
	current, err := s.currentSettings(ctx)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&easypourv1.UpdateSettingsResponse{Settings: current}), nil
}

func (s *EasyPourServer) saveSettings(ctx context.Context, msg *easypourv1.Settings) error {
	if s.settingsStore == nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("settings store unavailable"))
	}
	if msg == nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("settings required"))
	}
	if err := s.settingsStore.Update(ctx, &settings.Settings{AppriseURL: msg.GetAppriseUrl()}); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("update settings: %w", err))
	}
	return nil
}

func (s *EasyPourServer) currentSettings(ctx context.Context) (*easypourv1.Settings, error) {
	if s.settingsStore == nil {
		return &easypourv1.Settings{}, nil
	}
	cfg, err := s.settingsStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get settings: %w", err))
	}
	return &easypourv1.Settings{AppriseUrl: cfg.AppriseURL}, nil
}

// TestAppriseNotification sends a test notification to the Apprise API (admin only).
func (s *EasyPourServer) TestAppriseNotification(
	ctx context.Context,
	req *connect.Request[easypourv1.TestAppriseNotificationRequest],
) (*connect.Response[easypourv1.TestAppriseNotificationResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	url, err := s.resolveAppriseURL(ctx, req.Msg.GetAppriseUrl())
	if err != nil {
		return nil, err
	}
	payload := apprise.Payload{
		Title: "EasyPour test notification",
		Body:  "This is a test notification from EasyPour.",
		Type:  "info",
	}
	if err := apprise.Notify(s.webhookClient, url, payload); err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("apprise notify failed: %w", err))
	}
	return connect.NewResponse(&easypourv1.TestAppriseNotificationResponse{
		Message: "Test notification sent.",
	}), nil
}

// resolveAppriseURL prefers the URL in the request so admins can test before saving.
func (s *EasyPourServer) resolveAppriseURL(ctx context.Context, requested string) (string, error) {
	if url := strings.TrimSpace(requested); url != "" {
		return url, nil
	}
	saved, err := s.savedAppriseURL(ctx)
	if err != nil {
		return "", connect.NewError(connect.CodeInternal, fmt.Errorf("get settings: %w", err))
	}
	if saved == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("apprise URL required"))
	}
	return saved, nil
}

func (s *EasyPourServer) savedAppriseURL(ctx context.Context) (string, error) {
	if s.settingsStore == nil {
		return "", nil
	}
	cfg, err := s.settingsStore.Get(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(cfg.AppriseURL), nil
}
