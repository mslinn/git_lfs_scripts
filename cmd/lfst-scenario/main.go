package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/scenario"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

// Predefined scenarios based on gitScenarios.html
var scenarios = map[int]*scenario.Scenario{
	1:  {ID: 1, Name: "Bare repo - local", ServerType: "bare", Protocol: "local", GitServer: "bare"},
	2:  {ID: 2, Name: "Bare repo - SSH", ServerType: "bare", Protocol: "ssh", GitServer: "bare"},
	6:  {ID: 6, Name: "LFS Test Server - HTTP", ServerType: "lfs-test-server", Protocol: "http", GitServer: "bare", ServerURL: "http://gojira:8080"},
	7:  {ID: 7, Name: "LFS Test Server - HTTP/GitHub", ServerType: "lfs-test-server", Protocol: "http", GitServer: "github", ServerURL: "http://gojira:8080", RepoName: "mslinn/lfs-eval-test"},
	8:  {ID: 8, Name: "Giftless - local", ServerType: "giftless", Protocol: "local", GitServer: "bare"},
	9:  {ID: 9, Name: "Giftless - SSH", ServerType: "giftless", Protocol: "ssh", GitServer: "bare"},
	13: {ID: 13, Name: "Rudolfs - local", ServerType: "rudolfs", Protocol: "local", GitServer: "bare"},
	14: {ID: 14, Name: "Rudolfs - SSH", ServerType: "rudolfs", Protocol: "ssh", GitServer: "bare"},
}

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		force       bool
		dbPath      string
		workDir     string
		listOnly    bool
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.BoolVarP(&debug, "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.BoolVarP(&force, "force", "f", false, "Force recreation of existing repositories")
	pflag.StringVar(&dbPath, "db", "", "Path to SQLite database (default from config)")
	pflag.StringVar(&workDir, "work-dir", "/tmp/lfst", "Working directory for test execution")
	pflag.BoolVar(&listOnly, "list", false, "List available scenarios and exit")

	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-scenario version %s\n", version)
		os.Exit(0)
	}

	// Handle help
	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Handle list
	if listOnly {
		listScenarios()
		os.Exit(0)
	}

	// Get scenario ID
	args := pflag.Args()
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: scenario ID required\n\n")
		printUsage()
		os.Exit(1)
	}

	scenarioID, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid scenario ID '%s'\n", args[0])
		os.Exit(1)
	}

	// Get scenario
	scen, ok := scenarios[scenarioID]
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: scenario %d not found (use --list to see available scenarios)\n", scenarioID)
		os.Exit(1)
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

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Create and run scenario
	runner := scenario.NewRunner(scen, db, workDir, debug, force)
	if err := runner.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Scenario %d completed successfully\n", scenarioID)
	fmt.Printf("  Run ID: %d\n", runner.RunID)
	fmt.Printf("  View results: lfst-run show %d\n", runner.RunID)
}

func listScenarios() {
	fmt.Println("Available scenarios:")
	fmt.Println()
	fmt.Println("ID  Server             Protocol  Git Server  Description")
	fmt.Println("--  ------             --------  ----------  -----------")

	// Print in order
	ids := []int{1, 2, 6, 7, 8, 9, 13, 14}
	for _, id := range ids {
		scen := scenarios[id]
		fmt.Printf("%-3d %-18s %-9s %-11s %s\n",
			scen.ID,
			scen.ServerType,
			scen.Protocol,
			scen.GitServer,
			scen.Name,
		)
	}

	fmt.Println()
	fmt.Println("Note: Only scenarios 1, 2, 6-9, and 13-14 are currently implemented.")
	fmt.Println("      Additional scenarios require specific server configurations.")
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-scenario [OPTIONS] SCENARIO_ID\n\n")
	fmt.Fprintf(os.Stderr, "Run a complete Git LFS test scenario (all 7 steps)\n\n")
	pflag.PrintDefaults()
}

func printHelp() {
	fmt.Printf("lfst-scenario - Execute complete Git LFS test scenarios\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Executes a complete 7-step Git LFS evaluation scenario:\n")
	fmt.Printf("    1. Setup repository, configure LFS, copy initial files (~1.3GB)\n")
	fmt.Printf("    2. Add, commit, and push with timing measurements\n")
	fmt.Printf("    3. Modify, delete, and rename files\n")
	fmt.Printf("    4. Clone to second machine and verify checksums\n")
	fmt.Printf("    5. Make changes on second machine\n")
	fmt.Printf("    6. Pull changes back to first machine\n")
	fmt.Printf("    7. Untrack files from LFS\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-scenario [OPTIONS] SCENARIO_ID\n\n")

	fmt.Printf("OPTIONS:\n")
	pflag.PrintDefaults()

	fmt.Printf("\nEXAMPLES:\n")
	fmt.Printf("  # List available scenarios\n")
	fmt.Printf("  lfst-scenario --list\n\n")

	fmt.Printf("  # Run scenario 6 (LFS Test Server - HTTP)\n")
	fmt.Printf("  lfst-scenario 6\n\n")

	fmt.Printf("  # Run with debug output\n")
	fmt.Printf("  lfst-scenario -d 6\n\n")

	fmt.Printf("  # Use custom work directory\n")
	fmt.Printf("  lfst-scenario --work-dir /mnt/o/lfs_test 6\n\n")

	fmt.Printf("NOTES:\n")
	fmt.Printf("  - Requires ~2.4GB of test data (set LFS_TEST_DATA environment variable)\n")
	fmt.Printf("  - Work directory should have at least 5GB free space\n")
	fmt.Printf("  - For remote scenarios, requires passwordless SSH to gojira\n")
	fmt.Printf("  - Each run creates a test_run record in the database\n")
	fmt.Printf("  - All operations are timed with millisecond precision\n")
	fmt.Printf("  - Checksums are computed and stored for each step\n\n")
}
