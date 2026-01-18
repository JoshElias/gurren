// Package config combines input from env vars, config files and cli args
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	Auth    AuthConfig     `mapstructure:"auth"`
	Tunnels []TunnelConfig `mapstructure:"tunnels"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	Method  string `mapstructure:"method"`   // "auto", "agent", "publickey", "password"
	KeyPath string `mapstructure:"key_path"` // Optional: specific key path for publickey auth
}

// TunnelConfig defines a tunnel to a remote endpoint via an SSH host.
type TunnelConfig struct {
	Name   string `mapstructure:"name"`   // Friendly name for the tunnel
	Host   string `mapstructure:"host"`   // SSH host (from ~/.ssh/config or hostname)
	Remote string `mapstructure:"remote"` // Remote address (host:port)
	Local  string `mapstructure:"local"`  // Local bind address (host:port)
}

// Load reads configuration from file and environment.
// Config file locations (in order of precedence):
//  1. ./gurren.toml
//  2. ~/.config/gurren/config.toml
func Load() (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("auth.method", "auto")
	v.SetConfigType("toml")

	// Environment variables
	v.SetEnvPrefix("GURREN")
	v.AutomaticEnv()

	// Find config file in order of precedence:
	// 1. ~/.config/gurren/config.toml
	// 2. ~/gurren.toml
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to get home directory: %w", err)
	}

	configPaths := []string{
		filepath.Join(home, ".config", "gurren", "config.toml"),
		filepath.Join(home, "gurren.toml"),
	}

	var configFile string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			configFile = path
			break
		}
	}

	if configFile != "" {
		v.SetConfigFile(configFile)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}
	// No config file found is OK - use defaults

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &cfg, nil
}

// GetTunnelByName returns the tunnel config by name.
// Returns nil if not found.
func (c *Config) GetTunnelByName(name string) *TunnelConfig {
	for i := range c.Tunnels {
		if c.Tunnels[i].Name == name {
			return &c.Tunnels[i]
		}
	}
	return nil
}

// TunnelNames returns a list of all configured tunnel names.
func (c *Config) TunnelNames() []string {
	names := make([]string, len(c.Tunnels))
	for i, t := range c.Tunnels {
		names[i] = t.Name
	}
	return names
}
