package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	auth "github.com/jamesread/httpauthshim"
	"github.com/sirupsen/logrus"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/httpauthshim/providers/haslocal"
	"github.com/jamesread/httpauthshim/sessions"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/emptypb"

	"easypour/service/buildinfo"
	"easypour/service/internal/config"
	"easypour/service/internal/menu"
	"easypour/service/internal/order"
	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/gen/easypour/v1/easypourv1connect"
)

// orderEvent is sent to SSE clients on status update.
type orderEvent struct {
	Type    string `json:"type"`
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

// sseBroadcaster sends order status updates to all connected SSE clients.
type sseBroadcaster struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newSSEBroadcaster() *sseBroadcaster {
	return &sseBroadcaster{clients: make(map[chan []byte]struct{})}
}

func (b *sseBroadcaster) Subscribe() chan []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 8)
	b.clients[ch] = struct{}{}
	return ch
}

func (b *sseBroadcaster) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.clients, ch)
	close(ch)
}

func (b *sseBroadcaster) Broadcast(data []byte) {
	b.mu.Lock()
	clients := make([]chan []byte, 0, len(b.clients))
	for ch := range b.clients {
		clients = append(clients, ch)
	}
	b.mu.Unlock()
	for _, ch := range clients {
		select {
		case ch <- data:
		default:
		}
	}
}

const sseKeepaliveInterval = 30 * time.Second

// handleSSEOrderEvents serves GET /orders/events as text/event-stream; subscribes to broadcaster and sends keepalives every 30s.
func handleSSEOrderEvents(broadcast *sseBroadcaster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		ch := broadcast.Subscribe()
		defer broadcast.Unsubscribe(ch)
		ctx := r.Context()
		keepalive := time.NewTicker(sseKeepaliveInterval)
		defer keepalive.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case data, ok := <-ch:
				if !ok {
					return
				}
				if _, err := w.Write(append(append([]byte("data: "), data...), '\n', '\n')); err != nil {
					return
				}
				flusher.Flush()
			case <-keepalive.C:
				if _, err := w.Write([]byte(": keepalive\n\n")); err != nil {
					return
				}
				flusher.Flush()
			}
		}
	}
}

// EasyPourServer implements the EasyPourService
type EasyPourServer struct {
	easypourv1connect.UnimplementedEasyPourServiceHandler
	authCtx        *auth.AuthShimContext
	menuStore      *menu.Store
	orderStore     *order.Store
	webhooks       []config.Webhook
	webhookClient  *http.Client // skips TLS cert verification for webhook POSTs
	oauthProviders []*easypourv1.OAuthProvider // configured OAuth2 providers for login form
	sseBroadcast   *sseBroadcaster
}

// Init returns app version and configured OAuth2 providers. Callable unauthenticated.
func (s *EasyPourServer) Init(
	ctx context.Context,
	req *connect.Request[easypourv1.InitRequest],
) (*connect.Response[easypourv1.InitResponse], error) {
	resp := &easypourv1.InitResponse{Version: buildinfo.Version}
	if s.oauthProviders != nil {
		resp.OauthProviders = s.oauthProviders
	}
	return connect.NewResponse(resp), nil
}

// GetMenu returns the available drinks menu from the YAML store
func (s *EasyPourServer) GetMenu(
	ctx context.Context,
	req *connect.Request[easypourv1.GetMenuRequest],
) (*connect.Response[easypourv1.GetMenuResponse], error) {
	items, err := s.menuStore.Load()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load menu: %w", err))
	}
	return connect.NewResponse(&easypourv1.GetMenuResponse{Items: items}), nil
}

// webhookItem is a single item in the webhook payload's items array.
type webhookItem struct {
	MenuItemId  string `json:"menu_item_id"`
	Name        string `json:"name,omitempty"`
	AddSugar    bool   `json:"add_sugar"`
	AddMilk     bool   `json:"add_milk"`
	SugarAmount int32  `json:"sugar_amount"`
	MilkAmount  int32  `json:"milk_amount"`
}

// orderWebhookPayload is the JSON body sent to webhook URLs when an order is submitted.
type orderWebhookPayload struct {
	OrderId     string        `json:"order_id"`
	Status      string        `json:"status"`
	CreatedAt   int64         `json:"created_at"`
	OrderString string        `json:"order_string"`
	Items       []webhookItem `json:"items"`
}

// formatWebhookItemString returns a short description for one item, e.g. "Coffee (no sugar, no milk)" or "Espresso (2 sugars, milk)".
func formatWebhookItemString(name string, addSugar, addMilk bool, sugarAmount, milkAmount int32) string {
	sugarPart := "no sugar"
	if addSugar {
		if sugarAmount <= 0 {
			sugarAmount = 1
		}
		if sugarAmount == 1 {
			sugarPart = "1 sugar"
		} else {
			sugarPart = fmt.Sprintf("%d sugars", sugarAmount)
		}
	}
	milkPart := "no milk"
	if addMilk {
		milkPart = "milk"
	}
	return fmt.Sprintf("%s (%s, %s)", name, sugarPart, milkPart)
}

// getUsernameFromContext returns the authenticated username, or "" if auth disabled or guest.
func (s *EasyPourServer) getUsernameFromContext(ctx context.Context) string {
	if s.authCtx == nil {
		return ""
	}
	httpReq, _ := ctx.Value(httpRequestKey).(*http.Request)
	if httpReq == nil {
		return ""
	}
	user := s.authCtx.AuthFromHttpReq(httpReq)
	if user == nil || user.IsGuest() {
		return ""
	}
	return user.Username
}

// requireAuth returns the authenticated user or error. Used by RPCs that need any logged-in user.
func (s *EasyPourServer) requireAuth(ctx context.Context) (*authpublic.AuthenticatedUser, error) {
	if s.authCtx == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("auth required"))
	}
	httpReq, _ := ctx.Value(httpRequestKey).(*http.Request)
	if httpReq == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("request context missing"))
	}
	user := s.authCtx.AuthFromHttpReq(httpReq)
	if user == nil || user.IsGuest() {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	return user, nil
}

// OrderDrink places an order for a drink and persists it with status "pending".
func (s *EasyPourServer) OrderDrink(
	ctx context.Context,
	req *connect.Request[easypourv1.OrderRequest],
) (*connect.Response[easypourv1.OrderResponse], error) {
	orderReq := req.Msg

	orderID := uuid.New().String()
	username := s.getUsernameFromContext(ctx)

	o := &order.Order{
		ID:          orderID,
		MenuItemID:  orderReq.MenuItemId,
		Username:    username,
		AddSugar:    orderReq.AddSugar,
		AddMilk:     orderReq.AddMilk,
		SugarAmount: orderReq.SugarAmount,
		MilkAmount:  orderReq.MilkAmount,
		Status:      "pending",
	}
	if err := s.orderStore.Create(o); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("create order: %w", err))
	}

	response := &easypourv1.OrderResponse{
		OrderId:     o.ID,
		MenuItemId:  o.MenuItemID,
		AddSugar:    o.AddSugar,
		AddMilk:     o.AddMilk,
		SugarAmount: o.SugarAmount,
		MilkAmount:  o.MilkAmount,
		Status:      o.Status,
		CreatedAt:   o.CreatedAt,
	}

	logrus.Infof("Order received: %s - item %s (Sugar: %v, Milk: %v)",
		orderID, orderReq.MenuItemId, orderReq.AddSugar, orderReq.AddMilk)

	// Send order details to configured webhooks (fire-and-forget, with retries)
	nWebhooks := 0
	for _, wh := range s.webhooks {
		if wh.URL != "" {
			nWebhooks++
		}
	}
	if nWebhooks > 0 {
		logrus.Infof("Sending order %s to %d webhook(s)", orderID, nWebhooks)
		itemName := orderReq.MenuItemId
		if menuItems, err := s.menuStore.Load(); err == nil {
			for _, it := range menuItems {
				if it.Id == orderReq.MenuItemId {
					itemName = it.Name
					break
				}
			}
		}
		wi := webhookItem{
			MenuItemId:  orderReq.MenuItemId,
			Name:        itemName,
			AddSugar:    orderReq.AddSugar,
			AddMilk:     orderReq.AddMilk,
			SugarAmount: orderReq.SugarAmount,
			MilkAmount:  orderReq.MilkAmount,
		}
		orderString := formatWebhookItemString(itemName, orderReq.AddSugar, orderReq.AddMilk, orderReq.SugarAmount, orderReq.MilkAmount)
		payload := orderWebhookPayload{
			OrderId:     orderID,
			Status:      "pending",
			CreatedAt:   o.CreatedAt,
			OrderString: orderString,
			Items:       []webhookItem{wi},
		}
		body, _ := json.Marshal(payload)
		for _, wh := range s.webhooks {
			if wh.URL == "" {
				continue
			}
			url := wh.URL
			bodyCopy := make([]byte, len(body))
			copy(bodyCopy, body)
			go func() {
				s.postWebhookWithRetry(url, bodyCopy)
			}()
		}
	}

	return connect.NewResponse(response), nil
}

const (
	webhookTimeout   = 15 * time.Second
	webhookMaxRetries = 3
	webhookBaseDelay  = 500 * time.Millisecond
)

// postWebhookWithRetry POSTs order payload to url with exponential backoff. Logs errors.
func (s *EasyPourServer) postWebhookWithRetry(url string, body []byte) {
	var lastErr error
	for attempt := 0; attempt < webhookMaxRetries; attempt++ {
		if attempt > 0 {
			delay := webhookBaseDelay * (1 << (attempt - 1))
			time.Sleep(delay)
		}
		ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.webhookClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}
		lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	logrus.Warnf("Webhook %s failed after %d attempts: %v", url, webhookMaxRetries, lastErr)
}

// GetCurrentUser returns the authenticated user when auth is enabled (including is_admin for "admin" group),
// and the list of configured OAuth2 providers for the login form. Callable unauthenticated to obtain providers.
func (s *EasyPourServer) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[easypourv1.GetCurrentUserRequest],
) (*connect.Response[easypourv1.GetCurrentUserResponse], error) {
	resp := &easypourv1.GetCurrentUserResponse{}
	if s.oauthProviders != nil {
		resp.OauthProviders = s.oauthProviders
	}
	if s.authCtx == nil {
		return connect.NewResponse(resp), nil
	}
	httpReq, _ := ctx.Value(httpRequestKey).(*http.Request)
	if httpReq == nil {
		return connect.NewResponse(resp), nil
	}
	user := s.authCtx.AuthFromHttpReq(httpReq)
	if user != nil && !user.IsGuest() {
		resp.IsAuthenticated = true
		resp.Username = user.Username
		resp.IsAdmin = userInAdminGroup(user)
	}
	return connect.NewResponse(resp), nil
}

// userInAdminGroup returns true if the user's usergroups contain "admin"
func userInAdminGroup(user *authpublic.AuthenticatedUser) bool {
	for _, g := range strings.Split(user.UsergroupLine, ",") {
		if strings.TrimSpace(strings.ToLower(g)) == "admin" {
			return true
		}
	}
	return false
}

// requireAdmin returns a permission-denied error if the request user is not in the admin group.
func (s *EasyPourServer) requireAdmin(ctx context.Context) (*authpublic.AuthenticatedUser, error) {
	if s.authCtx == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("auth required"))
	}
	httpReq, _ := ctx.Value(httpRequestKey).(*http.Request)
	if httpReq == nil {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("request context missing"))
	}
	user := s.authCtx.AuthFromHttpReq(httpReq)
	if user == nil || user.IsGuest() {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("login required"))
	}
	if !userInAdminGroup(user) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}
	return user, nil
}

// CreateMenuItem adds a menu item (admin only)
func (s *EasyPourServer) CreateMenuItem(
	ctx context.Context,
	req *connect.Request[easypourv1.CreateMenuItemRequest],
) (*connect.Response[easypourv1.MenuItem], error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	item := req.Msg.GetItem()
	if item == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item required"))
	}
	created, err := s.menuStore.Create(item)
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
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	item := req.Msg.GetItem()
	if item == nil || item.Id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("item with id required"))
	}
	updated, err := s.menuStore.Update(item)
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
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	id := req.Msg.GetId()
	if id == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("id required"))
	}
	if err := s.menuStore.Delete(id); err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("delete menu item: %w", err))
	}
	return connect.NewResponse(&emptypb.Empty{}), nil
}

// GetOrder returns a single order by id. Caller must own the order or be admin.
func (s *EasyPourServer) GetOrder(
	ctx context.Context,
	req *connect.Request[easypourv1.GetOrderRequest],
) (*connect.Response[easypourv1.GetOrderResponse], error) {
	if _, err := s.requireAuth(ctx); err != nil {
		return nil, err
	}
	orderID := req.Msg.GetOrderId()
	if orderID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id required"))
	}
	o, err := s.orderStore.Get(orderID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get order: %w", err))
	}
	if o == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	username := s.getUsernameFromContext(ctx)
	if !userInAdminGroupFromContext(s.authCtx, ctx) && o.Username != username {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("not your order"))
	}
	return connect.NewResponse(&easypourv1.GetOrderResponse{Order: o.ToProto()}), nil
}

// ListOrders returns orders for the caller (own) or all orders if admin.
func (s *EasyPourServer) ListOrders(
	ctx context.Context,
	req *connect.Request[easypourv1.ListOrdersRequest],
) (*connect.Response[easypourv1.ListOrdersResponse], error) {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return nil, err
	}
	username := user.Username
	isAdmin := userInAdminGroup(user)
	list, err := s.orderStore.List(username, isAdmin)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("list orders: %w", err))
	}
	orders := make([]*easypourv1.Order, 0, len(list))
	for _, o := range list {
		orders = append(orders, o.ToProto())
	}
	return connect.NewResponse(&easypourv1.ListOrdersResponse{Orders: orders}), nil
}

var allowedStatuses = map[string]bool{"pending": true, "preparing": true, "delivered": true}

// validOrderStatusTransition returns true if the status change is allowed; pending→preparing is admin-only.
func validOrderStatusTransition(from, to string, isAdmin bool) bool {
	if !allowedStatuses[from] || !allowedStatuses[to] {
		return false
	}
	if from == to {
		return false
	}
	switch from {
	case "pending":
		if to == "preparing" {
			return isAdmin
		}
		return to == "delivered"
	case "preparing":
		return to == "delivered"
	case "delivered":
		return false
	default:
		return false
	}
}

// canUpdateOrderStatus returns true if the user can set the order to the new status (admin can do pending->preparing; user can only set delivered on own order).
func (s *EasyPourServer) canUpdateOrderStatus(ctx context.Context, o *order.Order, newStatus string) bool {
	user, err := s.requireAuth(ctx)
	if err != nil {
		return false
	}
	isAdmin := userInAdminGroup(user)
	if isAdmin {
		return validOrderStatusTransition(o.Status, newStatus, true)
	}
	if o.Username != user.Username {
		return false
	}
	return newStatus == "delivered" && validOrderStatusTransition(o.Status, "delivered", false)
}

// UpdateOrderStatus updates an order's status and broadcasts the change via SSE.
func (s *EasyPourServer) UpdateOrderStatus(
	ctx context.Context,
	req *connect.Request[easypourv1.UpdateOrderStatusRequest],
) (*connect.Response[easypourv1.UpdateOrderStatusResponse], error) {
	if _, err := s.requireAuth(ctx); err != nil {
		return nil, err
	}
	orderID := req.Msg.GetOrderId()
	status := strings.TrimSpace(strings.ToLower(req.Msg.GetStatus()))
	if orderID == "" || status == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("order_id and status required"))
	}
	if !allowedStatuses[status] {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("status must be pending, preparing, or delivered"))
	}
	o, err := s.orderStore.Get(orderID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get order: %w", err))
	}
	if o == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	if !s.canUpdateOrderStatus(ctx, o, status) {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("cannot update order status"))
	}
	updated, err := s.orderStore.UpdateStatus(orderID, status)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("update status: %w", err))
	}
	if updated == nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("order not found"))
	}
	ev := orderEvent{Type: "status_update", OrderID: updated.ID, Status: updated.Status}
	data, _ := json.Marshal(ev)
	s.sseBroadcast.Broadcast(data)
	return connect.NewResponse(&easypourv1.UpdateOrderStatusResponse{Order: updated.ToProto()}), nil
}

// userInAdminGroupFromContext returns true if the request user is in the admin group (for use when we already have ctx but not *AuthenticatedUser).
func userInAdminGroupFromContext(authCtx *auth.AuthShimContext, ctx context.Context) bool {
	if authCtx == nil {
		return false
	}
	httpReq, _ := ctx.Value(httpRequestKey).(*http.Request)
	if httpReq == nil {
		return false
	}
	user := authCtx.AuthFromHttpReq(httpReq)
	return user != nil && userInAdminGroup(user)
}

// context key for storing *http.Request (used by handlers that need auth identity)
type contextKey string

const httpRequestKey contextKey = "httpRequest"

// handleUpload accepts POST multipart/form-data with "image" file; requires admin. Saves to imagesDir, returns {"url": "/images/filename"}.
func handleUpload(authCtx *auth.AuthShimContext, imagesDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		user := authCtx.AuthFromHttpReq(r)
		if user == nil || user.IsGuest() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if !userInAdminGroup(user) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		if err := os.MkdirAll(imagesDir, 0755); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MiB
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "missing or invalid image field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		ext := filepath.Ext(header.Filename)
		if ext == "" {
			ext = ".jpg"
		}
		filename := uuid.New().String() + ext
		path := filepath.Join(imagesDir, filename)
		dest, err := os.Create(path)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer dest.Close()
		if _, err := io.Copy(dest, file); err != nil {
			os.Remove(path)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"url": "/images/" + filename})
	}
}

// withAuth wraps an http.Handler with httpauthshim authentication (session-based).
// Unauthenticated requests receive 401 without WWW-Authenticate so the browser does not show Basic auth.
// GetCurrentUser is allowed without auth so the login form can fetch the user and OAuth provider list.
func withAuth(authCtx *auth.AuthShimContext, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == easypourv1connect.EasyPourServiceInitProcedure ||
			r.URL.Path == easypourv1connect.EasyPourServiceGetCurrentUserProcedure {
			ctx := context.WithValue(r.Context(), httpRequestKey, r)
			h.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		user := authCtx.AuthFromHttpReq(r)
		if user == nil || user.IsGuest() {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), httpRequestKey, r)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// loginRequest is the JSON body for POST /login
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// loginOrSPA handles /login: POST goes to handleLogin; GET/HEAD when spa is set serves index.html
// so auth redirects to /login still load the SPA instead of 405.
func loginOrSPA(authCtx *auth.AuthShimContext, spa http.Handler) http.Handler {
	login := handleLogin(authCtx)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			login(w, r)
			return
		}
		if (r.Method == http.MethodGet || r.Method == http.MethodHead) && spa != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
			spa.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	})
}

// handleLogin handles POST /login for session-based auth. Validates username/password,
// creates a session, and sets the session cookie.
func handleLogin(authCtx *auth.AuthShimContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(req.Username)
		password := strings.TrimSpace(req.Password)
		if username == "" || password == "" {
			http.Error(w, "username and password required", http.StatusBadRequest)
			return
		}
		if !haslocal.CheckUserPassword(authCtx.Config, username, password) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		cfgUser := authCtx.Config.FindUserByUsername(username)
		usergroup := ""
		if cfgUser != nil {
			usergroup = cfgUser.UsergroupLine
		}
		sid := uuid.New().String()
		authCtx.RegisterUserSession("local", sid, username, strings.Split(usergroup, ",")...)
		cookieName := authCtx.Config.GetLocalSessionCookieName()
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    sid,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600, // 7 days
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"username": username})
	}
}

// handleLogout clears the session cookie so the client is logged out.
func handleLogout(authCtx *auth.AuthShimContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cookieName := authCtx.Config.GetLocalSessionCookieName()
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}
}

// getStaticDir returns EASYPOUR_STATIC_DIR if set and the directory exists; otherwise "".
func getStaticDir() string {
	dir := os.Getenv("EASYPOUR_STATIC_DIR")
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return ""
	}
	return abs
}

// spaFileServer serves files from root and falls back to index.html for GET/HEAD
// so SPA client-side routing works when the user navigates or refreshes.
func spaFileServer(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	rootAbs, _ := filepath.Abs(root)
	indexPath := filepath.Join(root, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		path := filepath.Clean(r.URL.Path)
		if path == "" || path == "." || path == "/" {
			// Serve index.html directly; do not use FileServer for it because
			// FileServer redirects /index.html to ./ (301), which breaks / and /./
			f, err := os.Open(indexPath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			fi, err := f.Stat()
			if err != nil || fi.IsDir() {
				http.NotFound(w, r)
				return
			}
			http.ServeContent(w, r, "index.html", fi.ModTime(), f)
			return
		}
		if path[0] != '/' {
			path = "/" + path
		}
		fpath := filepath.Join(root, path)
		fpathAbs, err := filepath.Abs(fpath)
		if err != nil || !strings.HasPrefix(fpathAbs+string(filepath.Separator), rootAbs+string(filepath.Separator)) {
			r = r.Clone(r.Context())
			r.URL.Path = "/index.html"
			fs.ServeHTTP(w, r)
			return
		}
		fi, err := os.Stat(fpath)
		if err == nil && fi.Mode().IsRegular() {
			fs.ServeHTTP(w, r)
			return
		}
		r = r.Clone(r.Context())
		r.URL.Path = "/index.html"
		fs.ServeHTTP(w, r)
	})
}

// setupAuth creates an AuthShimContext from the given config when auth is enabled.
// Returns nil when auth is disabled or misconfigured.
func setupAuth(appConfig *config.Config) (*auth.AuthShimContext, error) {
	if appConfig == nil || appConfig.Auth == nil {
		return nil, nil
	}
	authCfg := appConfig.Auth
	if !authCfg.LocalUsers.Enabled {
		return nil, nil
	}
	if len(authCfg.LocalUsers.Users) == 0 {
		logrus.Warn("Auth enabled but no users configured; auth disabled")
		return nil, nil
	}
	sessionStorage := sessions.NewSessionStorage(sessions.NewYAMLPersistence())
	authCtx, err := auth.NewAuthShimContext(authCfg, sessionStorage)
	if err != nil {
		return nil, err
	}
	authCtx.AddProvider(haslocal.CheckUserFromLocalSession)
	logrus.Info("Authentication enabled (httpauthshim, session-based login)")
	return authCtx, nil
}

func main() {
	hashPassword := flag.String("hash-password", "", "generate Argon2id hash for a password and exit (use in config.yaml auth.localUsers.users[].password)")
	configDir := flag.String("configdir", "", "directory containing config.yaml (for integration tests)")
	flag.Parse()
	if *configDir != "" {
		config.SetConfigDir(*configDir)
	}
	if *hashPassword != "" {
		hash, err := haslocal.CreateHash(*hashPassword)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hash-password failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(hash)
		os.Exit(0)
	}

	appCfg := config.LoadConfig()
	cfgPath := config.GetConfigPath()
	if cfgPath != "" {
		logrus.Infof("Config loaded from %s; %d webhook(s) configured", cfgPath, len(appCfg.Webhooks))
	} else {
		logrus.Infof("No config file found; using defaults (%d webhooks)", len(appCfg.Webhooks))
	}

	authCtx, err := setupAuth(appCfg)
	if err != nil {
		logrus.Fatalf("Setup auth failed: %v", err)
	}
	if authCtx != nil {
		defer func() {
			if err := authCtx.Shutdown(); err != nil {
				logrus.Warnf("Auth shutdown error: %v", err)
			}
		}()
	}

	menuPath := menu.GetMenuPath(cfgPath)
	menuStore := menu.NewStore(menuPath)
	items, err := menuStore.Load()
	if err != nil {
		logrus.Fatalf("Load menu: %v", err)
	}
	if _, statErr := os.Stat(menuPath); statErr != nil && os.IsNotExist(statErr) {
		if err := menuStore.Save(items); err != nil {
			logrus.Warnf("Write default menu: %v", err)
		} else {
			logrus.Infof("Created default menu at %s", menuPath)
		}
	}

	ordersDBPath := order.GetOrdersDBPath(cfgPath)
	orderStore, err := order.NewStore(ordersDBPath)
	if err != nil {
		logrus.Fatalf("Open order store: %v", err)
	}
	defer orderStore.Close()
	if err := orderStore.Init(); err != nil {
		logrus.Fatalf("Init order store: %v", err)
	}

	sseBroadcast := newSSEBroadcaster()

	webhookClient := &http.Client{
		Timeout: webhookTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var oauthProviders []*easypourv1.OAuthProvider
	for _, p := range appCfg.OAuthProviders {
		if p.ID != "" && p.Name != "" && p.AuthURL != "" {
			oauthProviders = append(oauthProviders, &easypourv1.OAuthProvider{
				Id:      p.ID,
				Name:    p.Name,
				AuthUrl: p.AuthURL,
			})
		}
	}
	server := &EasyPourServer{
		authCtx:        authCtx,
		menuStore:      menuStore,
		orderStore:     orderStore,
		webhooks:       appCfg.Webhooks,
		webhookClient:  webhookClient,
		oauthProviders: oauthProviders,
		sseBroadcast:   sseBroadcast,
	}
	mux := http.NewServeMux()
	staticDir := getStaticDir()
	var spaHandler http.Handler
	if staticDir != "" {
		spaHandler = spaFileServer(staticDir)
		logrus.Infof("Serving frontend from %s", staticDir)
	}
	if authCtx != nil {
		mux.Handle("/login", loginOrSPA(authCtx, spaHandler))
		mux.HandleFunc("/logout", handleLogout(authCtx))
		imagesDir := filepath.Join(filepath.Dir(menuPath), "images")
		mux.HandleFunc("/upload", handleUpload(authCtx, imagesDir))
		mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(imagesDir))))
		mux.Handle("/orders/events", withAuth(authCtx, handleSSEOrderEvents(sseBroadcast)))
	} else {
		mux.Handle("/orders/events", handleSSEOrderEvents(sseBroadcast))
	}
	path, handler := easypourv1connect.NewEasyPourServiceHandler(server)
	if authCtx != nil {
		mux.Handle(path, withAuth(authCtx, handler))
	} else {
		mux.Handle(path, handler)
	}
	if spaHandler != nil {
		mux.Handle("/", spaHandler)
	}

	addr := ":9654"
	logrus.Infof("Starting EasyPour service on %s", addr)
	logrus.Infof("ConnectRPC endpoint: http://localhost%s%s", addr, path)
	if authCtx != nil {
		logrus.Info("API requires authentication (HTTP Basic or configured providers)")
	}

	if err := http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		logrus.Fatalf("Server failed to start: %v", err)
	}
}
