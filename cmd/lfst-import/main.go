package main

import (
	"fmt"
	"io"
	"os"

	"github.com/lithammer/dedent"
	"github.com/mslinn/git_lfs_scripts/pkg/checksum"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		dbPath      string
		stdinMode   bool
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp,    "help",    "h", false, "Show this help message")
	pflag.BoolVarP(&debug,       "debug",   "d", false, "Enable debug output")
	pflag.BoolVarP(&debug,       "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.StringVar(&dbPath,     "db",      "",         "Path to SQLite database (default from config)")
	pflag.BoolVar(&stdinMode,    "stdin",        false, "Read JSON from stdin instead of file")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-import version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Use config database if not overridden
	if dbPath == "" {
		dbPath = cfg.GetDatabasePath()
	}

	if debug {
		fmt.Printf("Database: %s\n", dbPath)
	}

	// Get JSON input
	var jsonData []byte
	if stdinMode || len(pflag.Args()) == 0 {
		// Read from stdin
		if debug {
			fmt.Println("Reading JSON from stdin...")
		}
		jsonData, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Read from file
		jsonFile := pflag.Args()[0]
		if debug {
			fmt.Printf("Reading JSON from file: %s\n", jsonFile)
		}
		jsonData, err = os.ReadFile(jsonFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
			os.Exit(1)
		}
	}

	if len(jsonData) == 0 {
		fmt.Fprintf(os.Stderr, "Error: no JSON data provided\n")
		os.Exit(1)
	}

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Import checksums
	if err := checksum.ImportJSON(db, jsonData); err != nil {
		fmt.Fprintf(os.Stderr, "Error importing checksums: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Checksums imported successfully")
}

func printHelp() {
	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		lfst-import - Import checksum data from JSON into database

		Version: %s

		DESCRIPTION:
		  Imports checksum data from JSON format (exported by lfst-checksum)
		  into the SQLite database. Reads from stdin or a file.

		USAGE:
		  lfst-import [OPTIONS] [JSON_FILE]
		  lfst-import --stdin < checksums.json
		  cat checksums.json | lfst-import

		OPTIONS:
		`, version)))
	pflag.PrintDefaults()

	fmt.Print(dedent.Dedent(`

		EXAMPLES:
		  # Import from file
		  lfst-import checksums.json

		  # Import from stdin
		  cat checksums.json | lfst-import

		  # Import via SSH (typical remote usage)
		  cat checksums.json | ssh gojira lfst-import --stdin

		  # Custom database location
		  lfst-import --db /custom/path/test.db checksums.json

		CONFIGURATION:
		  Database path can be set via:
		  1. --db flag (highest priority)
		  2. LFS_TEST_DB environment variable
		  3. ~/.lfs-test-config file
		  4. Default: /home/mslinn/lfs_eval/lfs-test.db

		`))
}
