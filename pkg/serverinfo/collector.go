package serverinfo

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/timing"
)

// CollectedInfo contains all server information collected
type CollectedInfo struct {
	Info      *database.ServerInfo
	EnvVars   []*database.ServerEnvVar
	Processes []*database.ServerProcess
}

// CollectServerInfo collects server information from a remote host or local machine
func CollectServerInfo(serverURL string, runID int64, debug bool) (*CollectedInfo, error) {
	var hostname string
	var isLocal bool

	// Determine hostname from server URL
	if serverURL == "" {
		hostname = "localhost"
		isLocal = true
	} else {
		parsedURL, err := url.Parse(serverURL)
		if err != nil {
			return nil, fmt.Errorf("invalid server URL '%s': %w", serverURL, err)
		}
		hostname = parsedURL.Hostname()
		isLocal = (hostname == "localhost" || hostname == "127.0.0.1")
	}

	if debug {
		fmt.Printf("Collecting server information from %s...\n", hostname)
	}

	collectedAt := time.Now()
	collected := &CollectedInfo{
		Info:      &database.ServerInfo{RunID: runID, Hostname: hostname, CollectedAt: collectedAt},
		EnvVars:   []*database.ServerEnvVar{},
		Processes: []*database.ServerProcess{},
	}

	// Collect OS info
	if err := collectOSInfo(hostname, isLocal, collected.Info, debug); err != nil {
		if debug {
			fmt.Printf("  Warning: failed to collect OS info: %v\n", err)
		}
	}

	// Collect environment variables
	if err := collectEnvVars(hostname, isLocal, runID, collectedAt, &collected.EnvVars, debug); err != nil {
		if debug {
			fmt.Printf("  Warning: failed to collect env vars: %v\n", err)
		}
	}

	// Collect running processes
	if err := collectProcesses(hostname, isLocal, runID, collectedAt, &collected.Processes, debug); err != nil {
		if debug {
			fmt.Printf("  Warning: failed to collect processes: %v\n", err)
		}
	}

	if debug {
		fmt.Printf("  ✓ Collected: OS info, %d env vars, %d processes\n",
			len(collected.EnvVars), len(collected.Processes))
	}

	return collected, nil
}

// collectOSInfo collects operating system and kernel information
func collectOSInfo(hostname string, isLocal bool, info *database.ServerInfo, debug bool) error {
	var result *timing.Result

	// Get OS info (from /etc/os-release or similar)
	if isLocal {
		result = timing.Run("cat", []string{"/etc/os-release"}, nil)
	} else {
		result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "cat /etc/os-release"}, nil)
	}

	if result.ExitCode == 0 {
		// Extract OS name and version
		osInfo := parseOSRelease(result.Stdout)
		info.OSInfo = osInfo
	}

	// Get kernel version
	if isLocal {
		result = timing.Run("uname", []string{"-r"}, nil)
	} else {
		result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "uname -r"}, nil)
	}

	if result.ExitCode == 0 {
		info.KernelVersion = strings.TrimSpace(result.Stdout)
	}

	return nil
}

// parseOSRelease extracts useful information from /etc/os-release
func parseOSRelease(content string) string {
	lines := strings.Split(content, "\n")
	var name, version string

	for _, line := range lines {
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
		} else if strings.HasPrefix(line, "VERSION=") && version == "" {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION="), "\"")
		}
	}

	if name != "" {
		return name
	}
	if version != "" {
		return version
	}
	return "Unknown"
}

// collectEnvVars collects relevant environment variables
func collectEnvVars(hostname string, isLocal bool, runID int64, collectedAt time.Time, envVars *[]*database.ServerEnvVar, debug bool) error {
	var result *timing.Result

	// Get environment variables
	if isLocal {
		result = timing.Run("env", []string{}, nil)
	} else {
		result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "env"}, nil)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to get environment variables")
	}

	// Parse environment variables
	lines := strings.Split(result.Stdout, "\n")
	relevantVars := []string{
		"LFS_", "GIT_", "PATH", "HOME", "USER", "SHELL",
		"LANG", "LC_", "TERM", "SSH_", "DISPLAY",
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		varName := parts[0]
		varValue := parts[1]

		// Only store relevant variables
		isRelevant := false
		for _, prefix := range relevantVars {
			if strings.HasPrefix(varName, prefix) {
				isRelevant = true
				break
			}
		}

		if isRelevant {
			*envVars = append(*envVars, &database.ServerEnvVar{
				RunID:       runID,
				VarName:     varName,
				VarValue:    varValue,
				CollectedAt: collectedAt,
			})
		}
	}

	return nil
}

// collectProcesses collects running processes that might be relevant for testing
func collectProcesses(hostname string, isLocal bool, runID int64, collectedAt time.Time, processes *[]*database.ServerProcess, debug bool) error {
	var result *timing.Result

	// Get running processes
	if isLocal {
		result = timing.Run("ps", []string{"aux"}, nil)
	} else {
		result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "ps aux"}, nil)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("failed to get running processes")
	}

	// Parse ps output
	lines := strings.Split(result.Stdout, "\n")
	relevantProcesses := []string{
		"git", "lfs", "giftless", "rudolfs",
		"ssh", "sshd", "nginx", "apache",
		"docker", "containerd", "systemd",
	}

	// Skip header line
	for i, line := range lines {
		if i == 0 {
			continue
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse ps aux output (fields: USER PID %CPU %MEM VSZ RSS TTY STAT START TIME COMMAND)
		fields := regexp.MustCompile(`\s+`).Split(line, 11)
		if len(fields) < 11 {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		commandLine := fields[10]

		// Check if this is a relevant process
		isRelevant := false
		var processName string
		for _, pattern := range relevantProcesses {
			if strings.Contains(strings.ToLower(commandLine), pattern) {
				isRelevant = true
				processName = pattern
				break
			}
		}

		if isRelevant {
			pidCopy := pid
			*processes = append(*processes, &database.ServerProcess{
				RunID:       runID,
				PID:         &pidCopy,
				ProcessName: processName,
				CommandLine: commandLine,
				CollectedAt: collectedAt,
			})
		}
	}

	return nil
}

// StoreCollectedInfo stores collected server information in the database
func StoreCollectedInfo(db *database.DB, collected *CollectedInfo) error {
	// Store server info
	if err := db.CreateServerInfo(collected.Info); err != nil {
		return fmt.Errorf("failed to store server info: %w", err)
	}

	// Store environment variables
	for _, envVar := range collected.EnvVars {
		if err := db.CreateServerEnvVar(envVar); err != nil {
			return fmt.Errorf("failed to store env var %s: %w", envVar.VarName, err)
		}
	}

	// Store processes
	for _, process := range collected.Processes {
		if err := db.CreateServerProcess(process); err != nil {
			return fmt.Errorf("failed to store process %s: %w", process.ProcessName, err)
		}
	}

	return nil
}
