package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the LFS test configuration
type Config struct {
	Database   string `yaml:"database"`
	RemoteHost string `yaml:"remote_host"`
	AutoRemote bool   `yaml:"auto_remote"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Database:   "/home/mslinn/lfs_eval/lfs-test.db",
		RemoteHost: "gojira",
		AutoRemote: true,
	}
}

// Load loads configuration from file and environment variables
// Priority: environment variables > config file > defaults
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load from config file
	configPath := os.Getenv("LFS_TEST_CONFIG")
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(homeDir, ".lfs-test-config")
		}
	}

	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			// Config file is optional, so we just skip if not found
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("failed to load config file: %w", err)
			}
		}
	}

	// Override with environment variables
	if db := os.Getenv("LFS_TEST_DB"); db != "" {
		cfg.Database = db
	}
	if host := os.Getenv("LFS_REMOTE_HOST"); host != "" {
		cfg.RemoteHost = host
	}
	if autoRemote := os.Getenv("LFS_AUTO_REMOTE"); autoRemote != "" {
		cfg.AutoRemote = autoRemote == "true" || autoRemote == "1"
	}

	return cfg, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// Save saves the configuration to a file
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsRemoteHost returns true if the current hostname is not the remote host
func (cfg *Config) IsRemoteHost() bool {
	if !cfg.AutoRemote {
		return false
	}

	hostname, err := os.Hostname()
	if err != nil {
		return false
	}

	return hostname != cfg.RemoteHost
}

// GetDatabasePath returns the database path, expanding ~/ if needed
func (cfg *Config) GetDatabasePath() string {
	if len(cfg.Database) > 0 && cfg.Database[0] == '~' {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(homeDir, cfg.Database[2:])
		}
	}
	return cfg.Database
}
