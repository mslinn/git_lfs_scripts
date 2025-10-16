package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// Open opens or creates a SQLite database and initializes the schema
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	// WAL allows multiple readers while one writer is active
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set busy timeout to 5 seconds
	// If database is locked, retry for up to 5 seconds before failing
	if _, err := conn.Exec("PRAGMA busy_timeout=5000"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Enable foreign keys
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create schema
	if _, err := conn.Exec(schema); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// CreateTestRun creates a new test run record
func (db *DB) CreateTestRun(run *TestRun) error {
	result, err := db.conn.Exec(`
		INSERT INTO test_runs (scenario_id, server_type, protocol, git_server, started_at, status, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		run.ScenarioID, run.ServerType, run.Protocol, run.GitServer,
		run.StartedAt.Format(time.RFC3339), run.Status, run.Notes,
	)
	if err != nil {
		return fmt.Errorf("failed to create test run: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	run.ID = id
	return nil
}

// UpdateTestRun updates an existing test run
func (db *DB) UpdateTestRun(run *TestRun) error {
	var completedAt *string
	if run.CompletedAt != nil {
		t := run.CompletedAt.Format(time.RFC3339)
		completedAt = &t
	}

	_, err := db.conn.Exec(`
		UPDATE test_runs
		SET completed_at = ?, status = ?, notes = ?
		WHERE id = ?`,
		completedAt, run.Status, run.Notes, run.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update test run: %w", err)
	}

	return nil
}

// GetTestRun retrieves a test run by ID
func (db *DB) GetTestRun(id int64) (*TestRun, error) {
	var run TestRun
	var startedAt string
	var completedAt *string

	err := db.conn.QueryRow(`
		SELECT id, scenario_id, server_type, protocol, git_server, started_at, completed_at, status, notes
		FROM test_runs WHERE id = ?`, id,
	).Scan(
		&run.ID, &run.ScenarioID, &run.ServerType, &run.Protocol, &run.GitServer,
		&startedAt, &completedAt, &run.Status, &run.Notes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get test run: %w", err)
	}

	run.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
	if completedAt != nil {
		t, _ := time.Parse(time.RFC3339, *completedAt)
		run.CompletedAt = &t
	}

	return &run, nil
}

// ListTestRuns lists all test runs, optionally filtered by scenario ID (0 = all)
func (db *DB) ListTestRuns(scenarioID ...int) ([]*TestRun, error) {
	var query string
	var args []interface{}

	if len(scenarioID) > 0 && scenarioID[0] > 0 {
		query = `SELECT id, scenario_id, server_type, protocol, git_server, started_at, completed_at, status, notes
			FROM test_runs WHERE scenario_id = ? ORDER BY started_at DESC`
		args = append(args, scenarioID[0])
	} else {
		query = `SELECT id, scenario_id, server_type, protocol, git_server, started_at, completed_at, status, notes
			FROM test_runs ORDER BY started_at DESC`
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list test runs: %w", err)
	}
	defer rows.Close()

	var runs []*TestRun
	for rows.Next() {
		var run TestRun
		var startedAt string
		var completedAt *string

		err := rows.Scan(
			&run.ID, &run.ScenarioID, &run.ServerType, &run.Protocol, &run.GitServer,
			&startedAt, &completedAt, &run.Status, &run.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		run.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		if completedAt != nil {
			t, _ := time.Parse(time.RFC3339, *completedAt)
			run.CompletedAt = &t
		}

		runs = append(runs, &run)
	}

	return runs, nil
}

// CreateOperation creates a new operation record
func (db *DB) CreateOperation(op *Operation) error {
	result, err := db.conn.Exec(`
		INSERT INTO operations (run_id, step_number, operation, started_at, duration_ms, file_count, total_bytes, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		op.RunID, op.StepNumber, op.Operation,
		op.StartedAt.Format(time.RFC3339), op.DurationMs,
		op.FileCount, op.TotalBytes, op.Status, op.Error,
	)
	if err != nil {
		return fmt.Errorf("failed to create operation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	op.ID = id
	return nil
}

// ListOperations lists all operations for a test run
func (db *DB) ListOperations(runID int64) ([]*Operation, error) {
	rows, err := db.conn.Query(`
		SELECT id, run_id, step_number, operation, started_at, duration_ms, file_count, total_bytes, status, error
		FROM operations WHERE run_id = ? ORDER BY step_number, started_at`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	defer rows.Close()

	var ops []*Operation
	for rows.Next() {
		var op Operation
		var startedAt string

		err := rows.Scan(
			&op.ID, &op.RunID, &op.StepNumber, &op.Operation,
			&startedAt, &op.DurationMs, &op.FileCount, &op.TotalBytes,
			&op.Status, &op.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}

		op.StartedAt, _ = time.Parse(time.RFC3339, startedAt)
		ops = append(ops, &op)
	}

	return ops, nil
}

// CreateChecksum creates a new checksum record
func (db *DB) CreateChecksum(cs *Checksum) error {
	result, err := db.conn.Exec(`
		INSERT INTO checksums (run_id, step_number, file_path, crc32, size_bytes, computed_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		cs.RunID, cs.StepNumber, cs.FilePath, cs.CRC32, cs.SizeBytes,
		cs.ComputedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create checksum: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	cs.ID = id
	return nil
}

// ListChecksums lists all checksums for a test run and step
func (db *DB) ListChecksums(runID int64, stepNumber int) ([]*Checksum, error) {
	rows, err := db.conn.Query(`
		SELECT id, run_id, step_number, file_path, crc32, size_bytes, computed_at
		FROM checksums WHERE run_id = ? AND step_number = ? ORDER BY file_path`, runID, stepNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list checksums: %w", err)
	}
	defer rows.Close()

	var checksums []*Checksum
	for rows.Next() {
		var cs Checksum
		var computedAt string

		err := rows.Scan(
			&cs.ID, &cs.RunID, &cs.StepNumber, &cs.FilePath,
			&cs.CRC32, &cs.SizeBytes, &computedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan checksum: %w", err)
		}

		cs.ComputedAt, _ = time.Parse(time.RFC3339, computedAt)
		checksums = append(checksums, &cs)
	}

	return checksums, nil
}

// CreateRepositorySize creates a new repository size record
func (db *DB) CreateRepositorySize(rs *RepositorySize) error {
	result, err := db.conn.Exec(`
		INSERT INTO repository_sizes (run_id, step_number, location, size_bytes, file_count, measured_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		rs.RunID, rs.StepNumber, rs.Location, rs.SizeBytes, rs.FileCount,
		rs.MeasuredAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("failed to create repository size: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	rs.ID = id
	return nil
}

// ListRepositorySizes lists all repository sizes for a test run
func (db *DB) ListRepositorySizes(runID int64) ([]*RepositorySize, error) {
	rows, err := db.conn.Query(`
		SELECT id, run_id, step_number, location, size_bytes, file_count, measured_at
		FROM repository_sizes WHERE run_id = ? ORDER BY step_number, location`, runID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list repository sizes: %w", err)
	}
	defer rows.Close()

	var sizes []*RepositorySize
	for rows.Next() {
		var rs RepositorySize
		var measuredAt string

		err := rows.Scan(
			&rs.ID, &rs.RunID, &rs.StepNumber, &rs.Location,
			&rs.SizeBytes, &rs.FileCount, &measuredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repository size: %w", err)
		}

		rs.MeasuredAt, _ = time.Parse(time.RFC3339, measuredAt)
		sizes = append(sizes, &rs)
	}

	return sizes, nil
}

// GetChecksumsByRunAndStep retrieves all checksums for a specific run and step
func (db *DB) GetChecksumsByRunAndStep(runID int64, stepNumber int) ([]*Checksum, error) {
	return db.ListChecksums(runID, stepNumber)
}

// Rows wraps sql.Rows for use in query commands
type Rows = sql.Rows

// QueryRaw executes a raw SQL query and returns rows
func (db *DB) QueryRaw(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// QueryRowRaw executes a raw SQL query and returns a single row
func (db *DB) QueryRowRaw(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}
