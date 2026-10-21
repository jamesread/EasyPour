package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	auth "github.com/jamesread/httpauthshim"
	"github.com/jamesread/httpauthshim/providers/haslocal"
	"github.com/sirupsen/logrus"

	easypourv1 "easypour/service/gen/easypour/v1"
	"easypour/service/gen/easypour/v1/easypourv1connect"
	"easypour/service/internal/config"
	"easypour/service/internal/httpjson"
	"easypour/service/internal/menu"
	"easypour/service/internal/metrics"
	"easypour/service/internal/order"
	"easypour/service/internal/settings"
	"easypour/service/internal/sqlite"
)

// readHeaderTimeout guards against slow-header clients; no write timeout because /orders/events streams indefinitely.
const readHeaderTimeout = 10 * time.Second

type cliFlags struct {
	hashPassword string
	configDir    string
}

func parseFlags() cliFlags {
	hashPassword := flag.String("hash-password", "", "generate Argon2id hash for a password and exit (use in config.yaml auth.localUsers.users[].password)")
	configDir := flag.String("configdir", "", "directory containing config.yaml (for integration tests)")
	flag.Parse()
	return cliFlags{hashPassword: *hashPassword, configDir: *configDir}
}

func main() {
	flags := parseFlags()
	if flags.configDir != "" {
		config.SetConfigDir(flags.configDir)
	}
	if flags.hashPassword != "" {
		printPasswordHash(flags.hashPassword)
		return
	}
	if err := run(context.Background()); err != nil {
		logrus.Fatal(err)
	}
}

func printPasswordHash(password string) {
	hash, err := haslocal.CreateHash(password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash-password failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(hash)
}

func run(ctx context.Context) error {
	appCfg := loadConfig()
	authCtx, err := setupAuth(appCfg)
	if err != nil {
		return fmt.Errorf("setup auth failed: %w", err)
	}
	defer shutdownAuth(authCtx)
	return runWithAuth(ctx, appCfg, authCtx)
}

func runWithAuth(ctx context.Context, appCfg *config.Config, authCtx *auth.AuthShimContext) error {
	cfgPath := config.GetConfigPath()
	db, err := openDatabase(ctx, cfgPath)
	if err != nil {
		return err
	}
	defer closeDatabase(db)
	dataDir := sqlite.DataDir(cfgPath)
	server, err := newServer(ctx, appCfg, authCtx, db, dataDir)
	if err != nil {
		return err
	}
	return serve(newMux(server, authCtx, dataDir))
}

func loadConfig() *config.Config {
	appCfg := config.LoadConfig()
	if cfgPath := config.GetConfigPath(); cfgPath != "" {
		logrus.Infof("Config loaded from %s; %d webhook(s) configured", cfgPath, len(appCfg.Webhooks))
	} else {
		logrus.Infof("No config file found; using defaults (%d webhooks)", len(appCfg.Webhooks))
	}
	return appCfg
}

func openDatabase(ctx context.Context, cfgPath string) (*sql.DB, error) {
	db, err := sqlite.Open(ctx, sqlite.DBPath(cfgPath))
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := assertMigration(ctx, db); err != nil {
		closeDatabase(db)
		return nil, fmt.Errorf("database migration check: %w", err)
	}
	logDatabaseMigration(ctx, db)
	return db, nil
}

func closeDatabase(db *sql.DB) {
	if err := db.Close(); err != nil {
		logrus.Warnf("Close database: %v", err)
	}
}

// assertMigration fails startup unless config.RequiredMigration has been applied by sql-migrate.
func assertMigration(ctx context.Context, db *sql.DB) error {
	ok, err := sqlite.HasMigration(ctx, db, config.RequiredMigration)
	if err != nil {
		return fmt.Errorf("connected to the database, but the migrations table could not be queried. run sql-migrate up: %w", err)
	}
	if ok {
		return nil
	}
	latest, _ := sqlite.LatestMigration(ctx, db)
	if latest == "" {
		latest = "null"
	}
	return fmt.Errorf("requires database version %s but the database is at version %s; run database migrations", config.RequiredMigration, latest)
}

func logDatabaseMigration(ctx context.Context, db *sql.DB) {
	latest, err := sqlite.LatestMigration(ctx, db)
	if err != nil {
		logrus.Warnf("Could not read current database migration: %v", err)
		return
	}
	if latest == "" {
		latest = "none"
	}
	logrus.Infof("Database migration: %s", latest)
}

func newServer(ctx context.Context, appCfg *config.Config, authCtx *auth.AuthShimContext, db *sql.DB, dataDir string) (*EasyPourServer, error) {
	menuStore := menu.NewStore(db, dataDir)
	settingsStore := settings.NewStore(db)
	if err := prepareData(ctx, db, dataDir, menuStore, settingsStore); err != nil {
		return nil, err
	}
	return &EasyPourServer{
		authCtx:        authCtx,
		menuStore:      menuStore,
		orderStore:     order.NewStore(db),
		settingsStore:  settingsStore,
		webhooks:       appCfg.Webhooks,
		webhookClient:  newWebhookClient(),
		oauthProviders: oauthProvidersFromConfig(appCfg.OAuthProviders),
		sseBroadcast:   newSSEBroadcaster(),
	}, nil
}

func prepareData(ctx context.Context, db *sql.DB, dataDir string, menuStore *menu.Store, settingsStore *settings.Store) error {
	if err := sqlite.MigrateLegacy(ctx, db, dataDir); err != nil {
		return fmt.Errorf("migrate legacy databases: %w", err)
	}
	if err := menuStore.SeedIfEmpty(ctx); err != nil {
		return fmt.Errorf("seed menu: %w", err)
	}
	if err := settingsStore.EnsureDefaultCvars(ctx, "EasyPour"); err != nil {
		return fmt.Errorf("ensure default cvars: %w", err)
	}
	return nil
}

// newWebhookClient skips TLS verification because webhook and Apprise targets are commonly
// self-hosted on a LAN with self-signed certificates.
func newWebhookClient() *http.Client {
	return &http.Client{
		Timeout: httpjson.DefaultTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // see function comment
		},
	}
}

func oauthProvidersFromConfig(cfgs []config.OAuthProviderConfig) []*easypourv1.OAuthProvider {
	var providers []*easypourv1.OAuthProvider
	for _, p := range cfgs {
		if isCompleteOAuthProvider(p) {
			providers = append(providers, &easypourv1.OAuthProvider{Id: p.ID, Name: p.Name, AuthUrl: p.AuthURL})
		}
	}
	return providers
}

func isCompleteOAuthProvider(p config.OAuthProviderConfig) bool {
	return p.ID != "" && p.Name != "" && p.AuthURL != ""
}

func newMux(server *EasyPourServer, authCtx *auth.AuthShimContext, dataDir string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	spa := newSPAHandler()
	registerAuthRoutes(mux, authCtx, spa, dataDir)
	mux.Handle("/orders/events", requireSession(authCtx, server.sseBroadcast))
	path, handler := easypourv1connect.NewEasyPourServiceHandler(server)
	mux.Handle(path, requireSession(authCtx, handler))
	logrus.Infof("ConnectRPC endpoint: %s", path)
	if spa != nil {
		mux.Handle("/", spa)
	}
	return mux
}

func registerAuthRoutes(mux *http.ServeMux, authCtx *auth.AuthShimContext, spa http.Handler, dataDir string) {
	if authCtx == nil {
		return
	}
	imagesDir := filepath.Join(dataDir, "images")
	mux.Handle("/login", loginOrSPA(authCtx, spa))
	mux.HandleFunc("/logout", handleLogout(authCtx))
	mux.HandleFunc("/upload", handleUpload(authCtx, imagesDir))
	mux.Handle("/images/", http.StripPrefix("/images", http.FileServer(http.Dir(imagesDir))))
	logrus.Info("API requires authentication (HTTP Basic or configured providers)")
}

func serve(handler http.Handler) error {
	addr := config.ListenAddr()
	protocols := new(http.Protocols)
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		Protocols:         protocols,
	}
	logrus.Infof("Starting EasyPour service on %s", addr)
	return fmt.Errorf("server failed to start: %w", srv.ListenAndServe())
}
