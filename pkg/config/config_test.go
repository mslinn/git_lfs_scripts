package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if cfg.DatabasePath == "" {
		t.Error("DatabasePath should not be empty")
	}

	if cfg.RemoteHost != "gojira" {
		t.Errorf("Expected RemoteHost='gojira', got '%s'", cfg.RemoteHost)
	}

	if !cfg.AutoRemote {
		t.Error("AutoRemote should be true by default")
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create and save a config
	cfg := &Config{
		DatabasePath: "/tmp/test.db",
		RemoteHost:   "testhost",
		AutoRemote:   false,
	}

	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load the config back
	loadedCfg := DefaultConfig()
	if err := loadFromFile(loadedCfg, configPath); err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify values
	if loadedCfg.DatabasePath != cfg.DatabasePath {
		t.Errorf("DatabasePath mismatch: expected '%s', got '%s'", cfg.DatabasePath, loadedCfg.DatabasePath)
	}
	if loadedCfg.RemoteHost != cfg.RemoteHost {
		t.Errorf("RemoteHost mismatch: expected '%s', got '%s'", cfg.RemoteHost, loadedCfg.RemoteHost)
	}
	if loadedCfg.AutoRemote != cfg.AutoRemote {
		t.Errorf("AutoRemote mismatch: expected %v, got %v", cfg.AutoRemote, loadedCfg.AutoRemote)
	}
}

func TestLoadWithEnvironmentOverrides(t *testing.T) {
	// Save original environment
	origDB := os.Getenv("LFS_TEST_DB")
	origHost := os.Getenv("LFS_REMOTE_HOST")
	origAuto := os.Getenv("LFS_AUTO_REMOTE")
	origConfig := os.Getenv("LFS_TEST_CONFIG")
	defer func() {
		os.Setenv("LFS_TEST_DB", origDB)
		os.Setenv("LFS_REMOTE_HOST", origHost)
		os.Setenv("LFS_AUTO_REMOTE", origAuto)
		os.Setenv("LFS_TEST_CONFIG", origConfig)
	}()

	// Set environment variables
	os.Setenv("LFS_TEST_DB", "/env/test.db")
	os.Setenv("LFS_REMOTE_HOST", "envhost")
	os.Setenv("LFS_AUTO_REMOTE", "false")
	os.Setenv("LFS_TEST_CONFIG", "/nonexistent/config")

	// Load config (will use defaults + env overrides)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify environment overrides
	if cfg.DatabasePath != "/env/test.db" {
		t.Errorf("Expected DatabasePath from env '/env/test.db', got '%s'", cfg.DatabasePath)
	}
	if cfg.RemoteHost != "envhost" {
		t.Errorf("Expected RemoteHost from env 'envhost', got '%s'", cfg.RemoteHost)
	}
	if cfg.AutoRemote {
		t.Error("Expected AutoRemote to be false from env")
	}
}

func TestGetDatabasePath(t *testing.T) {
	tests := []struct {
		name     string
		dbPath   string
		expected func() string
	}{
		{
			name:   "absolute path",
			dbPath: "/absolute/path/to/db",
			expected: func() string {
				return "/absolute/path/to/db"
			},
		},
		{
			name:   "home directory expansion",
			dbPath: "~/lfs_eval/test.db",
			expected: func() string {
				home, _ := os.UserHomeDir()
				return filepath.Join(home, "lfs_eval/test.db")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{DatabasePath: tt.dbPath}
			got := cfg.GetDatabasePath()
			expected := tt.expected()
			if got != expected {
				t.Errorf("GetDatabasePath() = %v, want %v", got, expected)
			}
		})
	}
}

func TestIsRemoteHost(t *testing.T) {
	hostname, err := os.Hostname()
	if err != nil {
		t.Skip("Cannot get hostname, skipping test")
	}

	tests := []struct {
		name       string
		remoteHost string
		autoRemote bool
		expected   bool
	}{
		{
			name:       "same host with auto_remote",
			remoteHost: hostname,
			autoRemote: true,
			expected:   false, // Not remote (same host)
		},
		{
			name:       "different host with auto_remote",
			remoteHost: "different-host",
			autoRemote: true,
			expected:   true, // Is remote (different host)
		},
		{
			name:       "auto_remote disabled",
			remoteHost: "different-host",
			autoRemote: false,
			expected:   false, // Not remote (auto_remote disabled)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				RemoteHost: tt.remoteHost,
				AutoRemote: tt.autoRemote,
			}
			got := cfg.IsRemoteHost()
			if got != tt.expected {
				t.Errorf("IsRemoteHost() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	// Save original
	orig := os.Getenv("LFS_TEST_CONFIG")
	defer os.Setenv("LFS_TEST_CONFIG", orig)

	// Test with environment variable
	os.Setenv("LFS_TEST_CONFIG", "/custom/config/path")
	path := GetConfigPath()
	if path != "/custom/config/path" {
		t.Errorf("GetConfigPath() with env = %v, want /custom/config/path", path)
	}

	// Test without environment variable (should use home dir)
	os.Setenv("LFS_TEST_CONFIG", "")
	path = GetConfigPath()
	if path == "" {
		t.Error("GetConfigPath() should not return empty string")
	}
	if !filepath.IsAbs(path) && path != ".lfs-test-config" {
		t.Errorf("GetConfigPath() should return absolute path or relative fallback, got %v", path)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Try to save to a nested path that doesn't exist
	configPath := filepath.Join(tempDir, "nested", "dir", "config.yaml")

	cfg := DefaultConfig()
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("Save should create parent directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created in nested directory")
	}
}
