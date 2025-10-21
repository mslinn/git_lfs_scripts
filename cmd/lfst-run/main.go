package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/lithammer/dedent"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/scenario"
	"github.com/spf13/pflag"
)

var version = "dev" // Set by -ldflags during build

func main() {
	// Define global flags
	var (
		showVersion bool
		showHelp    bool
		debug       bool
		dbPath      string
	)

	pflag.BoolVarP(&showVersion, "version", "V", false, "Show version and exit")
	pflag.BoolVarP(&showHelp,    "help",    "h", false, "Show this help message")
	pflag.BoolVarP(&debug,       "debug",   "d", false, "Enable debug output")
	pflag.BoolVarP(&debug,       "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.StringVar(&dbPath, "db", "", "Path to SQLite database (default from config)")

	// Stop parsing at first non-flag argument (the subcommand)
	pflag.CommandLine.SetInterspersed(false)
	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-run version %s\n", version)
		os.Exit(0)
	}

	// Get subcommand
	args := pflag.Args()
	if len(args) == 0 || showHelp {
		printHelp()
		os.Exit(0)
	}

	subcommand := args[0]

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

	// Execute subcommand
	switch subcommand {
	case "create":
		handleCreate(db, args[1:], debug)
	case "list":
		handleList(db, args[1:], debug)
	case "show":
		handleShow(db, args[1:], debug)
	case "complete":
		handleComplete(db, args[1:], debug)
	case "fail":
		handleFail(db, args[1:], debug)
	case "update":
		handleUpdate(db, args[1:], debug)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func handleCreate(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("create", pflag.ExitOnError)
	scenarioID := fs.Int("scenario", 0, "Scenario ID (required)")
	notes      := fs.String("notes", "", "Optional notes about this test run")

	fs.Parse(args)

	// Validate required flags
	if *scenarioID == 0 {
		fmt.Fprintf(os.Stderr, "Error: --scenario is required (use 'lfst-scenario --list' to see available scenarios)\n")
		os.Exit(1)
	}

	// Get scenario details from centralized definitions
	scen, err := scenario.GetScenario(*scenarioID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintf(os.Stderr, "Use 'lfst-scenario --list' to see available scenarios\n")
		os.Exit(1)
	}

	// Create test run
	run := &database.TestRun{
		ScenarioID: *scenarioID,
		ServerType: scen.ServerType,
		Protocol:   scen.Protocol,
		GitServer:  scen.GitServer,
		StartedAt:  time.Now(),
		Status:     "running",
		Notes:      *notes,
	}

	err = db.CreateTestRun(run)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating test run: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created test run ID: %d\n", run.ID)
	if debug {
		notesLine := ""
		if *notes != "" {
			notesLine = fmt.Sprintf("  Notes: %s\n", *notes)
		}
		fmt.Print(dedent.Dedent(fmt.Sprintf(`
			  Scenario: %d (%s)
			  Server: %s
			  Protocol: %s
			  Git Server: %s
			  Status: running
			%s`, *scenarioID, scen.Name, scen.ServerType, scen.Protocol, scen.GitServer, notesLine)))
	}
}

func handleList(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("list", pflag.ExitOnError)
	status := fs.String("status", "", "Filter by status: running, completed, failed")
	limit := fs.Int("limit", 20, "Maximum number of runs to display")

	fs.Parse(args)

	runs, err := db.ListTestRuns()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing test runs: %v\n", err)
		os.Exit(1)
	}

	// Filter by status if specified
	if *status != "" {
		filtered := make([]*database.TestRun, 0)
		for _, run := range runs {
			if run.Status == *status {
				filtered = append(filtered, run)
			}
		}
		runs = filtered
	}

	// Apply limit
	if len(runs) > *limit {
		runs = runs[:*limit]
	}

	if len(runs) == 0 {
		fmt.Println("No test runs found")
		return
	}

	// Display as table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tScenario\tServer\tProtocol\tGit\tStatus\tStarted\tDuration\tNotes")
	fmt.Fprintln(w, "--\t--------\t------\t--------\t---\t------\t-------\t--------\t-----")

	for _, run := range runs {
		duration := "-"
		if run.CompletedAt != nil {
			d := run.CompletedAt.Sub(run.StartedAt)
			duration = fmt.Sprintf("%.1fs", d.Seconds())
		} else {
			d := time.Since(run.StartedAt)
			duration = fmt.Sprintf("%.1fs*", d.Seconds())
		}

		notes := run.Notes
		if len(notes) > 30 {
			notes = notes[:27] + "..."
		}

		fmt.Fprintf(w, "%d\t%d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			run.ID,
			run.ScenarioID,
			run.ServerType,
			run.Protocol,
			run.GitServer,
			run.Status,
			run.StartedAt.Format("15:04:05"),
			duration,
			notes,
		)
	}
	w.Flush()

	if debug {
		fmt.Printf("\nTotal runs: %d\n", len(runs))
	}
}

func handleShow(db *database.DB, args []string, _ bool) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: run ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-run show <RUN_ID>\n")
		os.Exit(1)
	}

	var runID int64
	if _, err := fmt.Sscanf(args[0], "%d", &runID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", args[0])
		os.Exit(1)
	}

	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", runID, err)
		os.Exit(1)
	}

	durationLine := ""
	if run.CompletedAt != nil {
		duration := run.CompletedAt.Sub(run.StartedAt)
		durationLine = fmt.Sprintf("  Completed:    %s\n  Duration:     %.2fs",
			run.CompletedAt.Format("2006-01-02 15:04:05"), duration.Seconds())
	} else {
		duration := time.Since(run.StartedAt)
		durationLine = fmt.Sprintf("  Running for:  %.2fs", duration.Seconds())
	}

	notesLine := ""
	if run.Notes != "" {
		notesLine = fmt.Sprintf("\n  Notes:        %s", run.Notes)
	}

	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		Test Run %d:
		  Scenario ID:  %d
		  Server Type:  %s
		  Protocol:     %s
		  Git Server:   %s
		  Status:       %s
		  Started:      %s
		%s%s
		`, run.ID, run.ScenarioID, run.ServerType, run.Protocol, run.GitServer,
		run.Status, run.StartedAt.Format("2006-01-02 15:04:05"), durationLine, notesLine)))
}

func handleComplete(db *database.DB, args []string, debug bool) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: run ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-run complete <RUN_ID> [--notes \"message\"]\n")
		os.Exit(1)
	}

	fs := pflag.NewFlagSet("complete", pflag.ExitOnError)
	notes := fs.String("notes", "", "Optional completion notes")
	fs.Parse(args[1:])

	var runID int64
	if _, err := fmt.Sscanf(args[0], "%d", &runID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", args[0])
		os.Exit(1)
	}

	// Get existing run
	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", runID, err)
		os.Exit(1)
	}

	// Update status
	now := time.Now()
	run.CompletedAt = &now
	run.Status = "completed"
	if *notes != "" {
		if run.Notes != "" {
			run.Notes += " | " + *notes
		} else {
			run.Notes = *notes
		}
	}

	if err := db.UpdateTestRun(run); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating test run: %v\n", err)
		os.Exit(1)
	}

	duration := now.Sub(run.StartedAt)
	fmt.Printf("✓ Test run %d marked as completed (%.2fs)\n", runID, duration.Seconds())
}

func handleFail(db *database.DB, args []string, debug bool) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: run ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-run fail <RUN_ID> [--notes \"error message\"]\n")
		os.Exit(1)
	}

	fs := pflag.NewFlagSet("fail", pflag.ExitOnError)
	notes := fs.String("notes", "", "Optional failure notes")
	fs.Parse(args[1:])

	var runID int64
	if _, err := fmt.Sscanf(args[0], "%d", &runID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", args[0])
		os.Exit(1)
	}

	// Get existing run
	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", runID, err)
		os.Exit(1)
	}

	// Update status
	now := time.Now()
	run.CompletedAt = &now
	run.Status = "failed"
	if *notes != "" {
		if run.Notes != "" {
			run.Notes += " | " + *notes
		} else {
			run.Notes = *notes
		}
	}

	if err := db.UpdateTestRun(run); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating test run: %v\n", err)
		os.Exit(1)
	}

	duration := now.Sub(run.StartedAt)
	fmt.Printf("✗ Test run %d marked as failed (%.2fs)\n", runID, duration.Seconds())
}

func handleUpdate(db *database.DB, args []string, debug bool) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Error: run ID required\n")
		fmt.Fprintf(os.Stderr, "Usage: lfst-run update <RUN_ID> [--notes \"message\"] [--status STATUS]\n")
		os.Exit(1)
	}

	fs := pflag.NewFlagSet("update", pflag.ExitOnError)
	notes := fs.String("notes", "", "Update notes")
	status := fs.String("status", "", "Update status: running, completed, failed")
	fs.Parse(args[1:])

	var runID int64
	if _, err := fmt.Sscanf(args[0], "%d", &runID); err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid run ID '%s'\n", args[0])
		os.Exit(1)
	}

	// Get existing run
	run, err := db.GetTestRun(runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", runID, err)
		os.Exit(1)
	}

	// Update fields
	updated := false
	if *notes != "" {
		if run.Notes != "" {
			run.Notes += " | " + *notes
		} else {
			run.Notes = *notes
		}
		updated = true
	}

	if *status != "" {
		validStatus := map[string]bool{
			"running":   true,
			"completed": true,
			"failed":    true,
		}
		if !validStatus[*status] {
			fmt.Fprintf(os.Stderr, "Error: invalid status '%s'\n", *status)
			fmt.Fprintf(os.Stderr, "Valid status: running, completed, failed\n")
			os.Exit(1)
		}
		run.Status = *status
		if *status != "running" && run.CompletedAt == nil {
			now := time.Now()
			run.CompletedAt = &now
		}
		updated = true
	}

	if !updated {
		fmt.Fprintf(os.Stderr, "Error: nothing to update (use --notes or --status)\n")
		os.Exit(1)
	}

	if err := db.UpdateTestRun(run); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating test run: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Test run %d updated\n", runID)
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-run [OPTIONS] COMMAND [ARGS...]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  create    Create a new test run\n")
	fmt.Fprintf(os.Stderr, "  list      List test runs\n")
	fmt.Fprintf(os.Stderr, "  show      Show details of a test run\n")
	fmt.Fprintf(os.Stderr, "  complete  Mark a test run as completed\n")
	fmt.Fprintf(os.Stderr, "  fail      Mark a test run as failed\n")
	fmt.Fprintf(os.Stderr, "  update    Update test run notes or status\n")
}

func printHelp() {
	fmt.Print(dedent.Dedent(fmt.Sprintf(`
		lfst-run - Manage Git LFS test run lifecycle

		Version: %s

		DESCRIPTION:
		  Create and manage test run records in the database. Each test run
		  represents one execution of a Git LFS test scenario.

		USAGE:
		  lfst-run [OPTIONS] COMMAND [ARGS...]

		COMMANDS:
		  create    Create a new test run
		  list      List test runs
		  show      Show details of a test run
		  complete  Mark a test run as completed
		  fail      Mark a test run as failed
		  update    Update test run notes or status

		GLOBAL OPTIONS:
		  -h, --help         Show this help message
		  -V, --version      Show version
		  -d, --debug        Enable debug output
		  -v, --verbose      Enable verbose output (alias for --debug)
		  --db PATH          Path to SQLite database

		EXAMPLES:
		  # List available scenarios
		  lfst-scenario --list

		  # Create a new test run for scenario 6 (LFS Test Server - HTTP)
		  lfst-run create --scenario 6

		  # Create with notes
		  lfst-run create --scenario 6 --notes "Testing HTTP push performance"

		  # List all running test runs
		  lfst-run list --status running

		  # Show details of test run 5
		  lfst-run show 5

		  # Mark test run 5 as completed
		  lfst-run complete 5 --notes "All tests passed"

		  # Mark test run 6 as failed
		  lfst-run fail 6 --notes "Push operation failed"

		For command-specific help:
		  lfst-run COMMAND --help

		`, version)))
}
