package database

const schema = `
CREATE TABLE IF NOT EXISTS test_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scenario_id INTEGER NOT NULL,
    server_type TEXT NOT NULL,
    protocol TEXT NOT NULL,
    git_server TEXT NOT NULL,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    status TEXT NOT NULL,
    notes TEXT
);

CREATE TABLE IF NOT EXISTS operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    step_number INTEGER NOT NULL,
    operation TEXT NOT NULL,
    started_at TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    file_count INTEGER,
    total_bytes INTEGER,
    status TEXT NOT NULL,
    error TEXT,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE TABLE IF NOT EXISTS checksums (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    step_number INTEGER NOT NULL,
    file_path TEXT NOT NULL,
    crc32 TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    computed_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE TABLE IF NOT EXISTS repository_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    step_number INTEGER NOT NULL,
    location TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    file_count INTEGER,
    measured_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE TABLE IF NOT EXISTS server_info (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    hostname TEXT NOT NULL,
    os_info TEXT,
    kernel_version TEXT,
    collected_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE TABLE IF NOT EXISTS server_env_vars (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    var_name TEXT NOT NULL,
    var_value TEXT,
    collected_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE TABLE IF NOT EXISTS server_processes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id INTEGER NOT NULL,
    pid INTEGER,
    process_name TEXT NOT NULL,
    command_line TEXT,
    collected_at TEXT NOT NULL,
    FOREIGN KEY (run_id) REFERENCES test_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_operations_run ON operations(run_id);
CREATE INDEX IF NOT EXISTS idx_checksums_run ON checksums(run_id);
CREATE INDEX IF NOT EXISTS idx_repo_sizes_run ON repository_sizes(run_id);
CREATE INDEX IF NOT EXISTS idx_test_runs_scenario ON test_runs(scenario_id);
CREATE INDEX IF NOT EXISTS idx_server_info_run ON server_info(run_id);
CREATE INDEX IF NOT EXISTS idx_server_env_vars_run ON server_env_vars(run_id);
CREATE INDEX IF NOT EXISTS idx_server_processes_run ON server_processes(run_id);
`
