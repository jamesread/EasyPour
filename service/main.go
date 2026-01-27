package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	auth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/jamesread/httpauthshim/providers/haslocal"
	"github.com/jamesread/httpauthshim/sessions"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/protobuf/types/known/emptypb"

	"easypour/service/internal/config"
	"easypour/service/internal/menu"
	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/gen/easypour/v1/easypourv1connect"
)

// EasyPourServer implements the EasyPourService
type EasyPourServer struct {
	easypourv1connect.UnimplementedEasyPourServiceHandler
	authCtx        *auth.AuthShimContext
	menuStore      *menu.Store
	webhooks       []config.Webhook
	webhookClient  *http.Client // skips TLS cert verification for webhook POSTs
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

// OrderDrink places an order for a drink
func (s *EasyPourServer) OrderDrink(
	ctx context.Context,
	req *connect.Request[easypourv1.OrderRequest],
) (*connect.Response[easypourv1.OrderResponse], error) {
	orderReq := req.Msg

	// Generate order ID
	orderID := uuid.New().String()
	createdAt := time.Now().Unix()

	// Create response
	response := &easypourv1.OrderResponse{
		OrderId:     orderID,
		MenuItemId:  orderReq.MenuItemId,
		AddSugar:    orderReq.AddSugar,
		AddMilk:     orderReq.AddMilk,
		SugarAmount: orderReq.SugarAmount,
		MilkAmount:  orderReq.MilkAmount,
		Status:      "preparing",
		CreatedAt:   createdAt,
	}

	// In a real implementation, you would queue this order for preparation
	log.Printf("Order received: %s - item %s (Sugar: %v, Milk: %v)",
		orderID, orderReq.MenuItemId, orderReq.AddSugar, orderReq.AddMilk)

	// Send order details to configured webhooks (fire-and-forget, with retries)
	nWebhooks := 0
	for _, wh := range s.webhooks {
		if wh.URL != "" {
			nWebhooks++
		}
	}
	if nWebhooks > 0 {
		log.Printf("Sending order %s to %d webhook(s)", orderID, nWebhooks)
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
			Status:      "preparing",
			CreatedAt:   createdAt,
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
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := s.webhookClient.Do(req)
		cancel()
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
	log.Printf("Webhook %s failed after %d attempts: %v", url, webhookMaxRetries, lastErr)
}

// GetCurrentUser returns the authenticated user when auth is enabled (including is_admin for "admin" group)
func (s *EasyPourServer) GetCurrentUser(
	ctx context.Context,
	req *connect.Request[easypourv1.GetCurrentUserRequest],
) (*connect.Response[easypourv1.GetCurrentUserResponse], error) {
	resp := &easypourv1.GetCurrentUserResponse{}
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
func withAuth(authCtx *auth.AuthShimContext, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		path := filepath.Clean(r.URL.Path)
		if path == "" || path == "." {
			path = "/"
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
		log.Print("Auth enabled but no users configured; auth disabled")
		return nil, nil
	}
	sessionStorage := sessions.NewSessionStorage(sessions.NewYAMLPersistence())
	authCtx, err := auth.NewAuthShimContext(authCfg, sessionStorage)
	if err != nil {
		return nil, err
	}
	authCtx.AddProvider(haslocal.CheckUserFromLocalSession)
	log.Print("Authentication enabled (httpauthshim, session-based login)")
	return authCtx, nil
}

func main() {
	hashPassword := flag.String("hash-password", "", "generate Argon2id hash for a password and exit (use in config.yaml auth.localUsers.users[].password)")
	flag.Parse()
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
		log.Printf("Config loaded from %s; %d webhook(s) configured", cfgPath, len(appCfg.Webhooks))
	} else {
		log.Printf("No config file found; using defaults (%d webhooks)", len(appCfg.Webhooks))
	}

	authCtx, err := setupAuth(appCfg)
	if err != nil {
		log.Fatalf("Setup auth failed: %v", err)
	}
	if authCtx != nil {
		defer func() {
			if err := authCtx.Shutdown(); err != nil {
				log.Printf("Auth shutdown error: %v", err)
			}
		}()
	}

	menuPath := menu.GetMenuPath(cfgPath)
	menuStore := menu.NewStore(menuPath)
	items, err := menuStore.Load()
	if err != nil {
		log.Fatalf("Load menu: %v", err)
	}
	if _, statErr := os.Stat(menuPath); statErr != nil && os.IsNotExist(statErr) {
		if err := menuStore.Save(items); err != nil {
			log.Printf("Write default menu: %v", err)
		} else {
			log.Printf("Created default menu at %s", menuPath)
		}
	}

	webhookClient := &http.Client{
		Timeout: webhookTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	server := &EasyPourServer{
		authCtx:       authCtx,
		menuStore:     menuStore,
		webhooks:      appCfg.Webhooks,
		webhookClient: webhookClient,
	}
	mux := http.NewServeMux()
	if authCtx != nil {
		mux.HandleFunc("/login", handleLogin(authCtx))
		imagesDir := filepath.Join(filepath.Dir(menuPath), "images")
		mux.HandleFunc("/upload", handleUpload(authCtx, imagesDir))
		mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(imagesDir))))
	}
	path, handler := easypourv1connect.NewEasyPourServiceHandler(server)
	if authCtx != nil {
		mux.Handle(path, withAuth(authCtx, handler))
	} else {
		mux.Handle(path, handler)
	}
	if staticDir := getStaticDir(); staticDir != "" {
		mux.Handle("/", spaFileServer(staticDir))
		log.Printf("Serving frontend from %s", staticDir)
	}

	addr := ":9654"
	log.Printf("Starting EasyPour service on %s", addr)
	log.Printf("ConnectRPC endpoint: http://localhost%s%s", addr, path)
	if authCtx != nil {
		log.Print("API requires authentication (HTTP Basic or configured providers)")
	}

	if err := http.ListenAndServe(
		addr,
		h2c.NewHandler(mux, &http2.Server{}),
	); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
