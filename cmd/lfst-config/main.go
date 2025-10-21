package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lithammer/dedent"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		configPath  string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.StringVar(&configPath, "config", "", "Path to config file (default: ~/.lfs-test-config)")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-config version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Get subcommand
	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: subcommand required\n\n")
		printUsage()
		os.Exit(1)
	}

	subcommand := args[0]

	// Override config path if specified
	if configPath != "" {
		os.Setenv("LFS_TEST_CONFIG", configPath)
	}

	// Execute subcommand
	switch subcommand {
	case "init":
		handleInit(args[1:])
	case "set":
		handleSet(args[1:])
	case "get":
		handleGet(args[1:])
	case "show":
		handleShow()
	case "path":
		handlePath()
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func handleInit(args []string) {
	// Parse flags for init
	var force bool
	flags := pflag.NewFlagSet("init", pflag.ExitOnError)
	flags.BoolVarP(&force, "force", "f", false, "Overwrite existing config file")
	flags.Parse(args)

	configPath := config.GetConfigPath()

	// Check if config exists
	if _, err := os.Stat(configPath); err == nil && !force {
		fmt.Fprintf(os.Stderr, "Error: config file already exists at %s\n", configPath)
		fmt.Fprintf(os.Stderr, "Use --force to overwrite\n")
		os.Exit(1)
	}

	// Create default config
	cfg := config.DefaultConfig()

	// Save config
	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Created config file at %s\n", configPath)
	fmt.Println("\nDefault configuration:")
	fmt.Printf("  database: %s\n", cfg.DatabasePath)
	fmt.Printf("  remote_host: %s\n", cfg.RemoteHost)
	fmt.Printf("  auto_remote: %v\n", cfg.AutoRemote)
	fmt.Println("\nEdit the file or use 'lfst-config set' to customize.")
}

func handleSet(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: 'set' requires KEY and VALUE arguments\n\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-config set KEY VALUE\n")
		fmt.Fprintf(os.Stderr, "\nValid keys:\n")
		fmt.Fprintf(os.Stderr, "  database      Path to SQLite database\n")
		fmt.Fprintf(os.Stderr, "  remote_host   Remote host for SSH operations\n")
		fmt.Fprintf(os.Stderr, "  auto_remote   Enable auto-remote detection (true/false)\n")
		os.Exit(1)
	}

	key := args[0]
	value := args[1]

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		fmt.Fprintf(os.Stderr, "Try running 'lfst-config init' first\n")
		os.Exit(1)
	}

	// Set the value
	switch key {
	case "database":
		cfg.DatabasePath = value
	case "remote_host":
		cfg.RemoteHost = value
	case "auto_remote":
		if value == "true" || value == "1" {
			cfg.AutoRemote = true
		} else if value == "false" || value == "0" {
			cfg.AutoRemote = false
		} else {
			fmt.Fprintf(os.Stderr, "Error: invalid value for auto_remote (use true/false or 1/0)\n")
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown config key '%s'\n", key)
		fmt.Fprintf(os.Stderr, "Valid keys: database, remote_host, auto_remote\n")
		os.Exit(1)
	}

	// Save config
	configPath := config.GetConfigPath()
	if err := cfg.Save(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Set %s = %v\n", key, value)
}

func handleGet(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Error: 'get' requires KEY argument\n\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-config get KEY\n")
		fmt.Fprintf(os.Stderr, "\nValid keys: database, remote_host, auto_remote\n")
		os.Exit(1)
	}

	key := args[0]

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Get the value
	switch key {
	case "database":
		fmt.Println(cfg.DatabasePath)
	case "remote_host":
		fmt.Println(cfg.RemoteHost)
	case "auto_remote":
		fmt.Println(cfg.AutoRemote)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown config key '%s'\n", key)
		fmt.Fprintf(os.Stderr, "Valid keys: database, remote_host, auto_remote\n")
		os.Exit(1)
	}
}

func handleShow() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to load config: %v\n", err)
		os.Exit(1)
	}

	configPath := config.GetConfigPath()
	fmt.Printf("Configuration from: %s\n\n", configPath)
	fmt.Printf("database:      %s\n", cfg.GetDatabasePath())
	fmt.Printf("remote_host:   %s\n", cfg.RemoteHost)
	fmt.Printf("auto_remote:   %v\n", cfg.AutoRemote)

	// Show environment variable overrides
	fmt.Println("\nEnvironment variable overrides:")
	if dbPath := os.Getenv("LFS_TEST_DB"); dbPath != "" {
		fmt.Printf("  LFS_TEST_DB=%s (overrides database)\n", dbPath)
	}
	if remoteHost := os.Getenv("LFS_REMOTE_HOST"); remoteHost != "" {
		fmt.Printf("  LFS_REMOTE_HOST=%s (overrides remote_host)\n", remoteHost)
	}
	if autoRemote := os.Getenv("LFS_AUTO_REMOTE"); autoRemote != "" {
		fmt.Printf("  LFS_AUTO_REMOTE=%s (overrides auto_remote)\n", autoRemote)
	}
}

func handlePath() {
	configPath := config.GetConfigPath()
	fmt.Println(configPath)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-config [OPTIONS] SUBCOMMAND\n\n")
	fmt.Fprintf(os.Stderr, "Manage LFS test configuration\n\n")
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  init          Create default config file\n")
	fmt.Fprintf(os.Stderr, "  set KEY VAL   Set configuration value\n")
	fmt.Fprintf(os.Stderr, "  get KEY       Get configuration value\n")
	fmt.Fprintf(os.Stderr, "  show          Show all configuration\n")
	fmt.Fprintf(os.Stderr, "  path          Show config file path\n\n")
	pflag.PrintDefaults()
}

func printHelp() {
	homeDir, _ := os.UserHomeDir()
	defaultDB := filepath.Join(homeDir, "lfs_eval", "lfs-test.db")

	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		lfst-config - Manage LFS test configuration

		Version: %s

		DESCRIPTION:
		  Manages configuration for LFS test commands. Configuration is stored in
		  ~/.lfs-test-config by default and can be overridden with environment variables.

		USAGE:
		  lfst-config [OPTIONS] SUBCOMMAND

		SUBCOMMANDS:
		  init          Create default configuration file
		  set KEY VAL   Set a configuration value
		  get KEY       Get a configuration value
		  show          Display all configuration values
		  path          Show the config file path

		CONFIGURATION KEYS:
		  database      Path to SQLite database
		                Default: /home/$USER/lfs_eval/lfs-test.db

		  remote_host   Remote host for SSH operations
		                Default: gojira

		  auto_remote   Automatically detect remote execution
		                Default: true

		ENVIRONMENT VARIABLES:
		  LFS_TEST_CONFIG    Path to config file
		  LFS_TEST_DB        Override database path
		  LFS_REMOTE_HOST    Override remote host
		  LFS_AUTO_REMOTE    Override auto_remote (true/false)

		OPTIONS:
		`, version)))
	pflag.PrintDefaults()

	fmt.Print(dedent.Dedent(fmt.Sprintf(`

		EXAMPLES:
		  # Create default config
		  lfst-config init

		  # Set custom database path
		  lfst-config set database /mnt/o/lfs-test.db

		  # Set remote host
		  lfst-config set remote_host myserver

		  # Disable auto-remote detection
		  lfst-config set auto_remote false

		  # View all configuration
		  lfst-config show

		  # Get specific value
		  lfst-config get database

		  # Find config file location
		  lfst-config path

		CONFIG FILE FORMAT:
		  # %s
		  database: %s
		  remote_host: gojira
		  auto_remote: true

		`, config.GetConfigPath(), defaultDB)))
}
