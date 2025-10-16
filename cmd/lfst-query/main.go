package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/mslinn/git_lfs_scripts/pkg/checksum"
	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
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
	pflag.BoolVarP(&showHelp, "help", "h", false, "Show this help message")
	pflag.BoolVarP(&debug, "debug", "d", false, "Enable debug output")
	pflag.BoolVarP(&debug, "verbose", "v", false, "Enable verbose output (alias for --debug)")
	pflag.StringVar(&dbPath, "db", "", "Path to SQLite database (default from config)")

	// Stop parsing at first non-flag argument (the subcommand)
	pflag.CommandLine.SetInterspersed(false)
	pflag.Parse()

	// Handle version
	if showVersion {
		fmt.Printf("lfst-query version %s\n", version)
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
	case "checksums":
		handleChecksums(db, args[1:], debug)
	case "compare":
		handleCompare(db, args[1:], debug)
	case "stats":
		handleStats(db, args[1:], debug)
	case "operations":
		handleOperations(db, args[1:], debug)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func handleChecksums(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("checksums", pflag.ExitOnError)
	runID := fs.Int64("run-id", 0, "Test run ID (required)")
	stepNumber := fs.Int("step", 0, "Step number (required)")
	limit := fs.Int("limit", 50, "Maximum number of checksums to display")

	fs.Parse(args)

	if *runID == 0 {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		os.Exit(1)
	}
	if *stepNumber == 0 {
		fmt.Fprintf(os.Stderr, "Error: --step is required\n")
		os.Exit(1)
	}

	checksums, err := db.GetChecksumsByRunAndStep(*runID, *stepNumber)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting checksums: %v\n", err)
		os.Exit(1)
	}

	if len(checksums) == 0 {
		fmt.Printf("No checksums found for run %d, step %d\n", *runID, *stepNumber)
		return
	}

	// Apply limit
	if len(checksums) > *limit {
		checksums = checksums[:*limit]
	}

	fmt.Printf("Checksums for run %d, step %d:\n\n", *runID, *stepNumber)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CRC32\tSize\tPath")
	fmt.Fprintln(w, "-----\t----\t----")

	for _, cs := range checksums {
		fmt.Fprintf(w, "%08x\t%s\t%s\n",
			cs.CRC32,
			checksum.FormatSize(cs.SizeBytes),
			cs.FilePath,
		)
	}
	w.Flush()

	if debug {
		fmt.Printf("\nTotal checksums: %d\n", len(checksums))
	}
}

func handleCompare(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("compare", pflag.ExitOnError)
	runID := fs.Int64("run-id", 0, "Test run ID (required)")
	fromStep := fs.Int("from", 0, "Source step number (required)")
	toStep := fs.Int("to", 0, "Target step number (required)")

	fs.Parse(args)

	if *runID == 0 {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		os.Exit(1)
	}
	if *fromStep == 0 {
		fmt.Fprintf(os.Stderr, "Error: --from is required\n")
		os.Exit(1)
	}
	if *toStep == 0 {
		fmt.Fprintf(os.Stderr, "Error: --to is required\n")
		os.Exit(1)
	}

	diffs, err := checksum.CompareChecksums(db, *runID, *fromStep, *toStep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error comparing checksums: %v\n", err)
		os.Exit(1)
	}

	if len(diffs) == 0 {
		fmt.Printf("No differences between step %d and step %d\n", *fromStep, *toStep)
		return
	}

	fmt.Printf("Changes from step %d to step %d:\n\n", *fromStep, *toStep)

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

func handleStats(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("stats", pflag.ExitOnError)
	runID := fs.Int64("run-id", 0, "Test run ID (0 = all runs)")

	fs.Parse(args)

	if *runID > 0 {
		// Stats for specific run
		run, err := db.GetTestRun(*runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: test run %d not found: %v\n", *runID, err)
			os.Exit(1)
		}

		fmt.Printf("Test Run %d Statistics:\n\n", *runID)
		fmt.Printf("  Scenario:     %d\n", run.ScenarioID)
		fmt.Printf("  Server:       %s\n", run.ServerType)
		fmt.Printf("  Protocol:     %s\n", run.Protocol)
		fmt.Printf("  Status:       %s\n", run.Status)

		// Count checksums per step
		rows, err := db.QueryRaw("SELECT step_number, COUNT(*) FROM checksums WHERE run_id = ? GROUP BY step_number ORDER BY step_number", *runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying checksums: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		fmt.Printf("\n  Checksums per step:\n")
		for rows.Next() {
			var step, count int
			if err := rows.Scan(&step, &count); err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
				continue
			}
			fmt.Printf("    Step %d: %d checksums\n", step, count)
		}

		// Count operations per step
		rows2, err := db.QueryRaw("SELECT step_number, COUNT(*), AVG(duration_ms) FROM operations WHERE run_id = ? GROUP BY step_number ORDER BY step_number", *runID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying operations: %v\n", err)
			os.Exit(1)
		}
		defer rows2.Close()

		fmt.Printf("\n  Operations per step:\n")
		for rows2.Next() {
			var step, count int
			var avgDuration float64
			if err := rows2.Scan(&step, &count, &avgDuration); err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
				continue
			}
			fmt.Printf("    Step %d: %d operations (avg %.1fms)\n", step, count, avgDuration)
		}

	} else {
		// Overall stats
		fmt.Printf("Overall Statistics:\n\n")

		// Count test runs by status
		rows, err := db.QueryRaw("SELECT status, COUNT(*) FROM test_runs GROUP BY status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying test runs: %v\n", err)
			os.Exit(1)
		}
		defer rows.Close()

		fmt.Printf("  Test runs by status:\n")
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
				continue
			}
			fmt.Printf("    %s: %d\n", status, count)
		}

		// Count test runs by server type
		rows2, err := db.QueryRaw("SELECT server_type, COUNT(*) FROM test_runs GROUP BY server_type")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error querying test runs: %v\n", err)
			os.Exit(1)
		}
		defer rows2.Close()

		fmt.Printf("\n  Test runs by server:\n")
		for rows2.Next() {
			var serverType string
			var count int
			if err := rows2.Scan(&serverType, &count); err != nil {
				fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
				continue
			}
			fmt.Printf("    %s: %d\n", serverType, count)
		}

		// Total checksums
		var totalChecksums int
		row := db.QueryRowRaw("SELECT COUNT(*) FROM checksums")
		if err := row.Scan(&totalChecksums); err != nil {
			fmt.Fprintf(os.Stderr, "Error counting checksums: %v\n", err)
		} else {
			fmt.Printf("\n  Total checksums: %d\n", totalChecksums)
		}

		// Total operations
		var totalOps int
		row2 := db.QueryRowRaw("SELECT COUNT(*) FROM operations")
		if err := row2.Scan(&totalOps); err != nil {
			fmt.Fprintf(os.Stderr, "Error counting operations: %v\n", err)
		} else {
			fmt.Printf("  Total operations: %d\n", totalOps)
		}
	}
}

func handleOperations(db *database.DB, args []string, debug bool) {
	fs := pflag.NewFlagSet("operations", pflag.ExitOnError)
	runID := fs.Int64("run-id", 0, "Test run ID (required)")
	stepNumber := fs.Int("step", 0, "Step number (0 = all steps)")
	limit := fs.Int("limit", 20, "Maximum number of operations to display")

	fs.Parse(args)

	if *runID == 0 {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		os.Exit(1)
	}

	var rows *database.Rows
	var err error

	if *stepNumber > 0 {
		rows, err = db.QueryRaw("SELECT step_number, operation_type, command, duration_ms, exit_code FROM operations WHERE run_id = ? AND step_number = ? ORDER BY timestamp", *runID, *stepNumber)
	} else {
		rows, err = db.QueryRaw("SELECT step_number, operation_type, command, duration_ms, exit_code FROM operations WHERE run_id = ? ORDER BY step_number, timestamp", *runID)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error querying operations: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Printf("Operations for run %d:\n\n", *runID)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "Step\tType\tCommand\tDuration\tExit")
	fmt.Fprintln(w, "----\t----\t-------\t--------\t----")

	count := 0
	for rows.Next() && count < *limit {
		var step, exitCode int
		var opType, command string
		var duration int64

		if err := rows.Scan(&step, &opType, &command, &duration, &exitCode); err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning row: %v\n", err)
			continue
		}

		// Truncate long commands
		if len(command) > 50 {
			command = command[:47] + "..."
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%dms\t%d\n",
			step, opType, command, duration, exitCode)
		count++
	}
	w.Flush()

	if debug {
		fmt.Printf("\nShowing %d operations\n", count)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lfst-query [OPTIONS] COMMAND [ARGS...]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  checksums    Show checksums for a specific run and step\n")
	fmt.Fprintf(os.Stderr, "  compare      Compare checksums between two steps\n")
	fmt.Fprintf(os.Stderr, "  stats        Show statistics about test runs\n")
	fmt.Fprintf(os.Stderr, "  operations   Show operations recorded for a test run\n")
}

func printHelp() {
	fmt.Printf("lfst-query - Query and report on Git LFS test data\n\n")
	fmt.Printf("Version: %s\n\n", version)
	fmt.Printf("DESCRIPTION:\n")
	fmt.Printf("  Query the test database to inspect checksums, compare steps,\n")
	fmt.Printf("  view operations, and generate statistics.\n\n")

	fmt.Printf("USAGE:\n")
	fmt.Printf("  lfst-query [OPTIONS] COMMAND [ARGS...]\n\n")

	fmt.Printf("COMMANDS:\n")
	fmt.Printf("  checksums    Show checksums for a specific run and step\n")
	fmt.Printf("  compare      Compare checksums between two steps\n")
	fmt.Printf("  stats        Show statistics about test runs\n")
	fmt.Printf("  operations   Show operations recorded for a test run\n\n")

	fmt.Printf("GLOBAL OPTIONS:\n")
	fmt.Printf("  -h, --help         Show this help message\n")
	fmt.Printf("  -V, --version      Show version\n")
	fmt.Printf("  -d, --debug        Enable debug output\n")
	fmt.Printf("  -v, --verbose      Enable verbose output (alias for --debug)\n")
	fmt.Printf("  --db PATH          Path to SQLite database\n\n")

	fmt.Printf("EXAMPLES:\n")
	fmt.Printf("  # Show checksums for run 5, step 1\n")
	fmt.Printf("  lfst-query checksums --run-id 5 --step 1\n\n")

	fmt.Printf("  # Compare checksums between step 1 and step 3\n")
	fmt.Printf("  lfst-query compare --run-id 5 --from 1 --to 3\n\n")

	fmt.Printf("  # Show statistics for test run 5\n")
	fmt.Printf("  lfst-query stats --run-id 5\n\n")

	fmt.Printf("  # Show overall database statistics\n")
	fmt.Printf("  lfst-query stats\n\n")

	fmt.Printf("  # Show operations for test run 5, step 2\n")
	fmt.Printf("  lfst-query operations --run-id 5 --step 2\n\n")

	fmt.Printf("For command-specific help:\n")
	fmt.Printf("  lfst-query COMMAND --help\n\n")
}
