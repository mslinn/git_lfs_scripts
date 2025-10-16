package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mslinn/git_lfs_scripts/pkg/checksum"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define flags
	var (
		showVersion  bool
		showHelp     bool
		debug        bool
		dbPath       string
		runID        int64
		stepNumber   int
		directory    string
		compareWith  int
		skipDatabase bool
		forceLocal   bool
		forceRemote  string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.BoolVarP(&debug, "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.StringVar(&dbPath, "db", "", "Path to SQLite database (default from config)")
	pflag.Int64Var(&runID, "run-id", 0, "Test run ID (required unless --skip-db)")
	pflag.IntVar(&stepNumber, "step", 0, "Step number (required unless --skip-db)")
	pflag.StringVar(&directory, "dir", ".", "Directory to compute checksums for")
	pflag.IntVar(&compareWith, "compare", 0, "Compare with checksums from this step number")
	pflag.BoolVar(&skipDatabase, "skip-db", false, "Skip database operations, just compute and display")
	pflag.BoolVar(&forceLocal, "local", false, "Force local database access (disable auto-remote)")
	pflag.StringVar(&forceRemote, "remote", "", "Force remote mode with specified host")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-checksum version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Validate flags
	if !skipDatabase {
		if runID == 0 {
			fmt.Fprintf(os.Stderr, "Error: --run-id is required (or use --skip-db)\n\n")
			printUsage()
			os.Exit(1)
		}
		if stepNumber == 0 {
			fmt.Fprintf(os.Stderr, "Error: --step is required (or use --skip-db)\n\n")
			printUsage()
			os.Exit(1)
		}
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

	// Determine if we should use remote mode
	useRemote := false
	remoteHost := ""

	if forceLocal {
		useRemote = false
	} else if forceRemote != "" {
		useRemote = true
		remoteHost = forceRemote
	} else if !skipDatabase && cfg.IsRemoteHost() {
		useRemote = true
		remoteHost = cfg.RemoteHost
	}

	// Get absolute path
	absDir, err := filepath.Abs(directory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get absolute path: %v\n", err)
		os.Exit(1)
	}

	// Check directory exists
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory does not exist: %s\n", absDir)
		os.Exit(1)
	}

	if debug {
		fmt.Printf("Computing checksums for: %s\n", absDir)
		if !skipDatabase {
			if useRemote {
				fmt.Printf("Remote mode: will send to %s:%s\n", remoteHost, dbPath)
			} else {
				fmt.Printf("Local mode: %s\n", dbPath)
			}
			fmt.Printf("Run ID: %d\n", runID)
			fmt.Printf("Step: %d\n", stepNumber)
		}
	}

	// Compute checksums
	checksums, err := checksum.ComputeDirectory(absDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing checksums: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Computed %d checksums\n", len(checksums))

	// Display checksums if debug or skip-db
	if debug || skipDatabase {
		for _, cs := range checksums {
			fmt.Printf("  %08x  %10s  %s\n",
				cs.CRC32,
				checksum.FormatSize(cs.SizeBytes),
				cs.Path,
			)
		}
	}

	// Skip database operations if requested
	if skipDatabase {
		os.Exit(0)
	}

	// Handle remote mode
	if useRemote {
		if err := executeRemote(remoteHost, dbPath, runID, stepNumber, checksums, debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error in remote mode: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Stored %d checksums on %s for step %d\n", len(checksums), remoteHost, stepNumber)

		// No comparison in remote mode (would need to fetch data back)
		if compareWith > 0 {
			fmt.Println("Note: --compare not supported in remote mode")
		}
		os.Exit(0)
	}

	// Local mode: open database directly
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify run exists
	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", runID, err)
		os.Exit(1)
	}

	if debug {
		fmt.Printf("Test run: scenario %d, server %s, protocol %s\n",
			run.ScenarioID, run.ServerType, run.Protocol)
	}

	// Store checksums in database
	if err := checksum.StoreChecksums(db, runID, stepNumber, checksums); err != nil {
		fmt.Fprintf(os.Stderr, "Error storing checksums: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Stored checksums in database for step %d\n", stepNumber)

	// Compare with previous step if requested
	if compareWith > 0 {
		fmt.Printf("\nComparing with step %d:\n", compareWith)
		diffs, err := checksum.CompareChecksums(db, runID, compareWith, stepNumber)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error comparing checksums: %v\n", err)
			os.Exit(1)
		}

		if len(diffs) == 0 {
			fmt.Println("  No differences found")
		} else {
			for _, diff := range diffs {
				switch diff.ChangeType {
				case "added":
					fmt.Printf("  ADDED:    %s (%s)\n",
						diff.FilePath, checksum.FormatSize(diff.NewSize))
				case "deleted":
					fmt.Printf("  DELETED:  %s (was %s)\n",
						diff.FilePath, checksum.FormatSize(diff.OldSize))
				case "modified":
					fmt.Printf("  MODIFIED: %s (%s)\n",
						diff.FilePath, checksum.FormatSize(diff.NewSize))
					if debug {
						fmt.Printf("            CRC: %s -> %s\n", diff.OldCRC32, diff.NewCRC32)
					}
				case "size-changed":
					fmt.Printf("  SIZE:     %s (%s -> %s)\n",
						diff.FilePath,
						checksum.FormatSize(diff.OldSize),
						checksum.FormatSize(diff.NewSize))
					if debug {
						fmt.Printf("            CRC: %s -> %s\n", diff.OldCRC32, diff.NewCRC32)
					}
				}
			}
			fmt.Printf("\nTotal differences: %d\n", len(diffs))
		}
	}
}

// executeRemote sends checksums to remote host via SSH
func executeRemote(host, dbPath string, runID int64, stepNumber int, checksums []*checksum.FileChecksum, debug bool) error {
	// Export to JSON
	jsonData, err := checksum.ExportJSON(runID, stepNumber, checksums)
	if err != nil {
		return fmt.Errorf("failed to export JSON: %w", err)
	}

	// Build SSH command
	sshCmd := fmt.Sprintf("lfst-import --stdin --db %s", dbPath)
	cmd := exec.Command("ssh", host, sshCmd)

	// Pipe JSON data to stdin
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH command: %w", err)
	}

	// Write JSON data
	if _, err := stdin.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write JSON data: %w", err)
	}
	stdin.Close()

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("SSH command failed: %w", err)
	}

	return nil
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-checksum [OPTIONS]\n\n")
	pflag.PrintDefaults()
}

func printHelp() {
	fmt.Printf("lfst-checksum - Compute and verify CRC32 checksums for Git LFS testing\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Computes CRC32 checksums for all files in a directory (recursively),\n")
	fmt.Printf("  stores them in a SQLite database, and optionally compares with checksums\n")
	fmt.Printf("  from a previous step to detect file changes.\n\n")
	fmt.Printf("  Files in .git/ directories and files named .checksums are automatically skipped.\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-checksum --run-id ID --step N --dir PATH\n")
	fmt.Printf("  lfst-checksum --run-id ID --step N --dir PATH --compare M\n")
	fmt.Printf("  lfst-checksum --skip-db --dir PATH\n")
	fmt.Printf("  lfst-checksum --local --run-id ID --step N --dir PATH\n")
	fmt.Printf("  lfst-checksum --remote HOST --run-id ID --step N --dir PATH\n\n")

	fmt.Printf("OPTIONS:\n")
	pflag.PrintDefaults()

	fmt.Printf("\nEXAMPLES:\n")
	fmt.Printf("  # Compute and display checksums without database\n")
	fmt.Printf("  lfst-checksum --skip-db --dir /path/to/repo\n\n")

	fmt.Printf("  # Store checksums for step 1 of test run 5\n")
	fmt.Printf("  lfst-checksum --run-id 5 --step 1 --dir /path/to/repo\n\n")

	fmt.Printf("  # Store checksums for step 3 and compare with step 1\n")
	fmt.Printf("  lfst-checksum --run-id 5 --step 3 --dir /path/to/repo --compare 1\n\n")

	fmt.Printf("  # Debug mode with verbose output\n")
	fmt.Printf("  lfst-checksum -d --run-id 5 --step 1 --dir /path/to/repo\n\n")

	fmt.Printf("REMOTE MODE:\n")
	fmt.Printf("  By default, lfst-checksum auto-detects if it's running on a remote machine\n")
	fmt.Printf("  (hostname != gojira) and automatically uses SSH to send data to the server.\n\n")
	fmt.Printf("  - --local: Force local mode (disable auto-remote)\n")
	fmt.Printf("  - --remote HOST: Force remote mode with specific host\n")
	fmt.Printf("  - Auto-remote can be disabled in ~/.lfs-test-config\n\n")

	fmt.Printf("CONFIGURATION:\n")
	fmt.Printf("  Configuration priority (highest to lowest):\n")
	fmt.Printf("  1. Command-line flags (--db, --remote, --local)\n")
	fmt.Printf("  2. Environment variables (LFS_TEST_DB, LFS_REMOTE_HOST)\n")
	fmt.Printf("  3. Config file (~/.lfs-test-config)\n")
	fmt.Printf("  4. Defaults (gojira, /home/mslinn/lfs_eval/lfs-test.db)\n\n")

	fmt.Printf("NOTES:\n")
	fmt.Printf("  - CRC32 values use the IEEE polynomial (same as cksum command)\n")
	fmt.Printf("  - Checksums are stored with millisecond-precision timestamps\n")
	fmt.Printf("  - The database file is created automatically if it doesn't exist\n")
	fmt.Printf("  - Use --skip-db for quick checksum verification without database\n")
	fmt.Printf("  - Remote mode requires passwordless SSH to the server\n\n")
}
