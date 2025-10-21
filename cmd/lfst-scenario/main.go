package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/lithammer/dedent"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/scenario"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define flags
	var (
		showVersion bool
		showHelp    bool
		verbose     bool
		force       bool
		dbPath      string
		workDir     string
		listOnly    bool
	)

	pflag.BoolVarP(&showVersion, "version",  "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp,    "help",     "h", false, "Show this help message")
	pflag.BoolVarP(&verbose,     "verbose",  "v", false, "Enable verbose output")
	pflag.BoolVarP(&force,       "force",    "f", false, "Force recreation of existing repositories")
	pflag.StringVar(&dbPath,     "db",       "", "Path to SQLite database (default from config)")
	pflag.StringVar(&workDir,    "work-dir", "/tmp/lfst", "Working directory for test execution")
	pflag.BoolVarP(&listOnly,    "list",     "L", false, "List available scenarios and exit")

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
	scen, err := scenario.GetScenario(scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v (use --list to see available scenarios)\n", err)
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
	runner := scenario.NewRunner(scen, db, workDir, verbose, force)
	if err := runner.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		✓ Scenario %d completed successfully
		  Run ID: %d
		  View results: lfst-run show %d
		`, scenarioID, runner.RunID, runner.RunID)))
}

func listScenarios() {
	scenarios := scenario.GetScenarios()
	fmt.Print(dedent.Dedent(`
		Available scenarios:

		ID  Server             Protocol  Git Server  Description
		--  ------             --------  ----------  -----------
		`))

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

	fmt.Print(dedent.Dedent(`
		Note: Only scenarios 1, 2, 6-9, and 13-14 are currently implemented.
		      Additional scenarios require specific server configurations.
		`))
}

func printUsage() {
	fmt.Fprint(os.Stderr, dedent.Dedent(`
		Usage: lfst-scenario [OPTIONS] SCENARIO_ID

		Run a complete Git LFS test scenario (all 7 steps)

		`))
	pflag.PrintDefaults()
}

func printHelp() {
	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		lfst-scenario - Execute complete Git LFS test scenarios

		Version: %s

		DESCRIPTION:
		  Executes a complete 7-step Git LFS evaluation scenario:
		    1. Setup repository, configure LFS, copy initial files (~1.3GB)
		    2. Add, commit, and push with timing measurements
		    3. Modify, delete, and rename files
		    4. Clone to second machine and verify checksums
		    5. Make changes on second machine
		    6. Pull changes back to first machine
		    7. Untrack files from LFS

		USAGE:
		  lfst-scenario [OPTIONS] SCENARIO_ID

		OPTIONS:
		`, version)))
	pflag.PrintDefaults()

	fmt.Print(dedent.Dedent(`

		EXAMPLES:
		  # List available scenarios
		  lfst-scenario --list

		  # Run scenario 6 (LFS Test Server - HTTP)
		  lfst-scenario 6

		  # Run with verbose output
		  lfst-scenario -v 6

		  # Use custom work directory
		  lfst-scenario --work-dir /mnt/o/lfs_test 6

		NOTES:
		  - Requires ~2.4GB of test data (set LFS_TEST_DATA environment variable)
		  - Work directory should have at least 5GB free space
		  - For remote scenarios, requires passwordless SSH to gojira
		  - Each run creates a test_run record in the database
		  - All operations are timed with millisecond precision
		  - Checksums are computed and stored for each step

		`))
}
