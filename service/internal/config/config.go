package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jamesread/httpauthshim/authpublic"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

const defaultListenAddr = ":9654"

// RequiredMigration is the sql-migrate id this binary expects to be applied.
const RequiredMigration = "0.base.sql"

// ListenAddr returns the bind address: $PORT if set, otherwise :9654.
// PORT may be a bare port ("8080") or a full address (":8080" / "0.0.0.0:8080").
func ListenAddr() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		return normalizePort(port)
	}
	return defaultListenAddr
}

func normalizePort(port string) string {
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

// Webhook is a single webhook target with a URL.
type Webhook struct {
	URL string `yaml:"url"`
}

// OAuthProviderConfig describes a configured OAuth2 provider for the login form.
type OAuthProviderConfig struct {
	ID      string `yaml:"id"`       // provider id, e.g. "google", "github"
	Name    string `yaml:"name"`     // display name, e.g. "Google", "GitHub"
	AuthURL string `yaml:"auth_url"` // URL to start OAuth2 flow
}

// Config holds application configuration including auth (httpauthshim format).
type Config struct {
	ConfigVersion  int                    `yaml:"configVersion"`
	Auth           *authpublic.Config     `yaml:"auth"`
	Webhooks       []Webhook              `yaml:"webhooks"`
	OAuthProviders []OAuthProviderConfig `yaml:"oauthProviders"`
}

var configDirOverride string

// SetConfigDir sets the directory containing config.yaml (e.g. for integration tests).
// GetConfigPath will then look for config.yaml in this directory first.
func SetConfigDir(dir string) {
	configDirOverride = dir
}

// GetConfigPath returns the first existing config file path, or empty string.
func GetConfigPath() string {
	if configDirOverride != "" {
		p := filepath.Join(configDirOverride, "config.yaml")
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	candidates := []string{
		"./config.yaml",
		"./config/config.yaml",
		os.Getenv("EASYPOUR_CONFIG_FILE"),
	}
	for _, p := range candidates {
		if p == "" {
			continue
		}
		abs, _ := filepath.Abs(p)
		if abs != "" {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// LoadConfig loads config using koanf from file or returns defaults.
func LoadConfig() *Config {
	cfg := &Config{ConfigVersion: 1}
	path := GetConfigPath()
	if path == "" {
		return cfg
	}
	k := koanf.New(".")
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		return cfg
	}
	if err := k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return cfg
	}
	return cfg
}
