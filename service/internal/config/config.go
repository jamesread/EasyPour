package config

import (
	"os"
	"path/filepath"

	"github.com/jamesread/httpauthshim/authpublic"
	"gopkg.in/yaml.v3"
)

// Webhook is a single webhook target with a URL.
type Webhook struct {
	URL string `yaml:"url"`
}

// Config holds application configuration including auth (httpauthshim format).
type Config struct {
	ConfigVersion int                `yaml:"configVersion"`
	Auth          *authpublic.Config `yaml:"auth"`
	Webhooks      []Webhook          `yaml:"webhooks"`
}

// GetConfigPath returns the first existing config file path, or empty string.
func GetConfigPath() string {
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

// LoadConfig loads config from file or returns defaults.
func LoadConfig() *Config {
	cfg := &Config{ConfigVersion: 1}
	path := GetConfigPath()
	if path == "" {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return cfg
	}
	return cfg
}
