// Test program to verify report generation
package main

import (
	"fmt"
	"os"

	"github.com/mslinn/git_lfs_scripts/pkg/config"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/report"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test-report.go <run_id>")
		fmt.Println("\nExample: go run test-report.go 2")
		os.Exit(1)
	}

	runID := os.Args[1]

	// Load config to get database path
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	dbPath := cfg.GetDatabasePath()
	fmt.Printf("Using database: %s\n", dbPath)

	// Open database
	db, err := database.Open(dbPath)
	if err != nil {
		fmt.Printf("Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Parse run ID
	var id int64
	fmt.Sscanf(runID, "%d", &id)

	// Generate report
	fmt.Printf("\nGenerating report for run ID %d...\n\n", id)
	gen := report.NewGenerator(db, id, fmt.Sprintf("/tmp/test-report-%d.md", id))
	if err := gen.Generate(); err != nil {
		fmt.Printf("Failed to generate report: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Report generated successfully!")
}
