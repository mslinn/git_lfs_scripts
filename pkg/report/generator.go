package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mslinn/git_lfs_scripts/pkg/database"
)

// Generator generates test run reports
type Generator struct {
	DB     *database.DB
	RunID  int64
	Output string // Output file path (optional)
}

// NewGenerator creates a new report generator
func NewGenerator(db *database.DB, runID int64, outputPath string) *Generator {
	return &Generator{
		DB:     db,
		RunID:  runID,
		Output: outputPath,
	}
}

// Generate generates a complete report and displays it on console
func (g *Generator) Generate() error {
	// Fetch all data
	run, err := g.DB.GetTestRun(g.RunID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	serverInfo, err := g.DB.GetServerInfo(g.RunID)
	if err != nil {
		return fmt.Errorf("failed to get server info: %w", err)
	}

	envVars, err := g.DB.ListServerEnvVars(g.RunID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	processes, err := g.DB.ListServerProcesses(g.RunID)
	if err != nil {
		return fmt.Errorf("failed to get processes: %w", err)
	}

	operations, err := g.DB.ListOperations(g.RunID)
	if err != nil {
		return fmt.Errorf("failed to get operations: %w", err)
	}

	// Generate markdown content
	markdown := g.generateMarkdown(run, serverInfo, envVars, processes, operations)

	// Display on console
	fmt.Println(markdown)

	// Write to file if output path is specified
	if g.Output != "" {
		if err := g.writeToFile(markdown); err != nil {
			return fmt.Errorf("failed to write report: %w", err)
		}
		fmt.Printf("\nReport saved to: %s\n", g.Output)
	}

	return nil
}

// generateMarkdown generates markdown content for the report
func (g *Generator) generateMarkdown(
	run *database.TestRun,
	serverInfo *database.ServerInfo,
	envVars []*database.ServerEnvVar,
	processes []*database.ServerProcess,
	operations []*database.Operation,
) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# Git LFS Test Run Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %d\n\n", run.ID))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("---\n\n")

	// Test Run Information
	sb.WriteString("## Test Run Information\n\n")
	sb.WriteString(fmt.Sprintf("- **Scenario ID:** %d\n", run.ScenarioID))
	sb.WriteString(fmt.Sprintf("- **Server Type:** %s\n", run.ServerType))
	sb.WriteString(fmt.Sprintf("- **Protocol:** %s\n", run.Protocol))
	sb.WriteString(fmt.Sprintf("- **Git Server:** %s\n", run.GitServer))
	sb.WriteString(fmt.Sprintf("- **Status:** %s\n", run.Status))
	sb.WriteString(fmt.Sprintf("- **Started:** %s\n", run.StartedAt.Format(time.RFC3339)))
	if run.CompletedAt != nil {
		sb.WriteString(fmt.Sprintf("- **Completed:** %s\n", run.CompletedAt.Format(time.RFC3339)))
		duration := run.CompletedAt.Sub(run.StartedAt)
		sb.WriteString(fmt.Sprintf("- **Duration:** %s\n", duration.Round(time.Second)))
	}
	if run.Notes != "" {
		sb.WriteString(fmt.Sprintf("- **Notes:** %s\n", run.Notes))
	}
	sb.WriteString("\n")

	// Server Information
	if serverInfo != nil {
		sb.WriteString("## Server Information\n\n")
		sb.WriteString(fmt.Sprintf("- **Hostname:** %s\n", serverInfo.Hostname))
		if serverInfo.OSInfo != "" {
			sb.WriteString(fmt.Sprintf("- **OS:** %s\n", serverInfo.OSInfo))
		}
		if serverInfo.KernelVersion != "" {
			sb.WriteString(fmt.Sprintf("- **Kernel:** %s\n", serverInfo.KernelVersion))
		}
		sb.WriteString(fmt.Sprintf("- **Collected:** %s\n", serverInfo.CollectedAt.Format(time.RFC3339)))
		sb.WriteString("\n")
	}

	// Environment Variables
	if len(envVars) > 0 {
		sb.WriteString("## Server Environment Variables\n\n")
		sb.WriteString("| Variable | Value |\n")
		sb.WriteString("|----------|-------|\n")
		for _, ev := range envVars {
			// Truncate long values
			value := ev.VarValue
			if len(value) > 80 {
				value = value[:77] + "..."
			}
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` |\n", ev.VarName, value))
		}
		sb.WriteString("\n")
	}

	// Running Processes
	if len(processes) > 0 {
		sb.WriteString("## Server Processes\n\n")
		sb.WriteString("| PID | Name | Command Line |\n")
		sb.WriteString("|-----|------|-------------|\n")
		for _, p := range processes {
			pid := "N/A"
			if p.PID != nil {
				pid = fmt.Sprintf("%d", *p.PID)
			}
			// Truncate long command lines
			cmdLine := p.CommandLine
			if len(cmdLine) > 100 {
				cmdLine = cmdLine[:97] + "..."
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | `%s` |\n", pid, p.ProcessName, cmdLine))
		}
		sb.WriteString("\n")
	}

	// Operations
	if len(operations) > 0 {
		sb.WriteString("## Operations\n\n")
		sb.WriteString("| Step | Operation | Duration (ms) | Status | Files | Bytes |\n")
		sb.WriteString("|------|-----------|---------------|--------|-------|-------|\n")

		for _, op := range operations {
			files := "N/A"
			if op.FileCount != nil {
				files = fmt.Sprintf("%d", *op.FileCount)
			}
			bytes := "N/A"
			if op.TotalBytes != nil {
				bytes = formatBytes(*op.TotalBytes)
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %d | %s | %s | %s |\n",
				op.StepNumber, op.Operation, op.DurationMs, op.Status, files, bytes))
		}
		sb.WriteString("\n")
	}

	// Footer
	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by lfst-scenario test harness*\n")

	return sb.String()
}

// writeToFile writes markdown content to a file
func (g *Generator) writeToFile(content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(g.Output)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(g.Output, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
