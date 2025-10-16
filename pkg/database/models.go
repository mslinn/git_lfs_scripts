package database

import "time"

// TestRun represents a complete test run for a scenario
type TestRun struct {
	ID          int64
	ScenarioID  int
	ServerType  string // 'lfs-test-server', 'giftless', 'rudolfs', 'bare'
	Protocol    string // 'http', 'https', 'ssh', 'local'
	GitServer   string // 'bare', 'github'
	StartedAt   time.Time
	CompletedAt *time.Time
	Status      string // 'running', 'completed', 'failed'
	Notes       string
}

// Operation represents a timed Git/LFS operation
type Operation struct {
	ID         int64
	RunID      int64
	StepNumber int
	Operation  string // 'add', 'commit', 'push', 'pull', 'clone', 'lfs-track', etc.
	StartedAt  time.Time
	DurationMs int64 // Millisecond precision
	FileCount  *int
	TotalBytes *int64
	Status     string // 'success', 'failed'
	Error      string
}

// Checksum represents a file CRC32 checksum
type Checksum struct {
	ID         int64
	RunID      int64
	StepNumber int
	FilePath   string
	CRC32      string
	SizeBytes  int64
	ComputedAt time.Time
}

// RepositorySize represents storage metrics
type RepositorySize struct {
	ID         int64
	RunID      int64
	StepNumber int
	Location   string // 'client-git', 'client-lfs', 'server-git', 'server-lfs'
	SizeBytes  int64
	FileCount  *int
	MeasuredAt time.Time
}
