# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build, Test, Run Commands

### Building

```bash
make build              # Build all commands to bin/
make build-checksum     # Build only lfst-checksum
make clean              # Remove built binaries
make install            # Install to /usr/local/bin (requires sudo)
```

### Testing

```bash
make test                  # Run all tests
make test-coverage         # Run tests with coverage report
make test-race             # Run tests with race detector
go test -v ./pkg/checksum  # Test specific package
```

### Code Quality

```bash
make fmt               # Format Go code
make vet               # Run go vet
make check             # Run fmt, vet, and test together
```

### Running Commands

All commands are built to `bin/` directory.
Version is injected during build from `VERSION` file.

```bash
bin/lfst-scenario --list                    # List available scenarios
bin/lfst-scenario 1                         # Run scenario 1
bin/lfst-scenario 6 --verbose --work-dir /tmp/test6
bin/lfst-run list                           # List test runs
bin/lfst-run show --run 123                 # Show details of run 123
bin/lfst-query checksums --run 123 --step 3 # Query checksums for run
```

## Architecture Overview

### Command Structure

Six CLI commands live in `cmd/`, each is a standalone Go program:

- **lfst-scenario**: Executes full 7-step LFS test scenarios (main automation tool)
- **lfst-run**: Manages test run records in database (create, list, show, complete, fail, update)
- **lfst-query**: Queries test results and checksums from database
- **lfst-checksum**: Computes and compares file checksums
- **lfst-config**: Manages configuration (show, init)
- **lfst-import**: Imports test data

All commands use `spf13/pflag` and follow consistent flag patterns (`-h/-V/-d/-v/-q`).

### Package Structure (`pkg/`)

Core packages provide reusable functionality:

- **scenario**: Scenario definitions (map of ID to server/protocol/git server configs) and Runner that executes the 7-step test workflow. Each step method (Step1_Setup through Step7_Untrack) is self-contained and records operations to database.

- **git**: Context-based git operations wrapper. All operations use `timing.Run()` to measure performance and `recordOperation()` to save to database. Critical: operations need repo directory passed explicitly since they run in different working directories.

- **database**: SQLite wrapper with schema for test_runs, operations, checksums, and timings. Uses WAL mode for concurrency. Models are in models.go, schema SQL in schema.go.

- **checksum**: CRC32 checksum computation for file verification. ComputeDirectory() walks trees, CompareChecksums() finds differences between steps.

- **testdata**: Test data location and file copying logic. Handles local and remote (rsync) paths. Returns file descriptors mapping source to dest.

- **timing**: Command execution with timing metrics. Returns Result with duration, exit code, stdout/stderr. Used by all git operations.

- **config**: YAML configuration management. Config path precedence: --config flag > .lfs-test-config > ~/.lfs-test-config > /etc/lfs-test-config

### Key Architectural Patterns

**7-Step Scenario Flow** (`pkg/scenario/scenario.go`):
Each scenario executes 7 steps that test LFS functionality:

1. Setup: Init repo, configure LFS tracking (*.pdf, *.mov, etc), copy 1.3GB test files, compute checksums
2. Initial Push: Add/commit/push all files with timing
3. Modifications: Update files with v2, delete some (video1.m4v, video4.ogg), rename zip2.zip, commit/push
4. Second Clone: Clone to repo2 dir, verify checksums match step 3
5. Second Client Push: Create README.md in repo2, commit/push
6. First Client Pull: Pull changes back to original repo
7. Untrack: Remove LFS tracking, migrate files back to regular git

Each step records checksums to database for verification.

**Database-Backed Operations** (`pkg/database/database.go`):
All operations record timing and status. Test runs track: scenario_id, server_type, protocol, git_server, status (running/completed/failed), timestamps. Operations table links to test_runs via run_id, records each git operation (clone, commit, push, etc) with timing data.

**Context-Based Git Operations** (`pkg/git/operations.go`):
Context struct carries DB, RunID, StepNumber, Debug, WorkDir through operation chains. Each operation (Clone, InitRepo, Commit, etc) uses timing.Run(), records to database, and returns formatted errors. Debug mode prints step numbers and timing info.

## Critical Coding Requirements

### Git Operations - ALWAYS USE -C FLAG

Git operations run in specific working directories. The timing.Run() helper passes working directory as third argument, but when constructing git commands manually, ALWAYS use `-C` option:

```go
// Correct - specify working directory explicitly
result := timing.Run("git", []string{"-C", repoDir, "status"}, nil)

// Correct - use env/working dir in timing.Run
result := timing.Run("git", []string{"status"}, &timing.Options{WorkDir: repoDir})

// Incorrect - will fail in wrong directory
result := timing.Run("git", []string{"status"}, nil)
```

### Command Line Standards

- Use `spf13/pflag` (POSIX-style) not standard `flag` package
- All commands must support: `-h/--help`, `-V/--version`, `-d/--debug`, `-v/--verbose`, `-q/--quiet`
- Version is set via `-ldflags "-X main.version=$(VERSION)"` during build (see Makefile)
- Use multiline strings with `github.com/lithammer/dedent` for help text alignment
- Subcommands use `pflag.NewFlagSet()` for local flag parsing (see lfst-run for example)

### Testing Requirements

- Write unit tests for complex logic rather than reasoning through it
- Write methods and functions so they are testable (dependency injection, pure functions)
- Test all non-trivial methods and classes
- Test files live next to source (e.g., `checksum_test.go` next to `checksum.go`)

### Code Documentation

- Every exported method/function needs inline documentation
- Document what the function does, not how (the code shows how)
- Include parameter meanings and return value semantics
- Example: `// ComputeDirectory calculates CRC32 checksums for all files in a directory tree`

### Git Commit Practices

- Make atomic git commits with clear messages for every action
- Commit message format: "Verb noun details" (e.g., "Fix unused debug parameter warnings in lfst-run")
- Include context in commit body when behavior changes

## Project Context

### Authoritative Specification

The website articles are the authoritative spec (code is aspirational and may not match):

- **Public**: `https://mslinn.com/git/index.html` (section: "Git Large File System")
- **Dev server**: `http://localhost:4001/git/index.html`
- **Website source**: `/var/sitesUbuntu/www.mslinn.com/collections/_git`
- **Scenarios template**: `/var/sitesUbuntu/www.mslinn.com/_includes/gitScenarios.html`

Read website articles for context. Website text is more accurate than code.

### Scenario Definitions

Scenarios are defined in `pkg/scenario/scenario.go` GetScenarios() map and correspond to gitScenarios.html:

- ID 1,2: Bare repo (local/SSH)
- ID 6,7: LFS Test Server (HTTP with bare/GitHub)
- ID 8,9: Giftless (local/SSH)
- ID 13,14: Rudolfs (local/SSH)

Each scenario specifies ServerType, Protocol, and GitServer. ServerURL points to remote LFS server (e.g., "http://gojira:8080" for test server on gojira machine).

### Development Workflow

- Multi-machine testing: `gojira` (192.168.1.183) is primary test server; `bear` and `camille` are clients
- Work in `claude` branch for both code and website
- Project locations:
  - `bear`: `/mnt/f/work/git/git_lfs_scripts`
  - `camille`: `/mnt/d/work/git/git_lfs_scripts`
  - `gojira`: `/work/git/git_lfs_scripts`

### Authorization Phases

Get approval before proceeding to next steps.

**Current - Step 2**: Test all LFS servers and progressively build test harness and reporting mechanism in Go.

**Step 3** (not authorized): Update website articles to explain plan at medium detail level with consistency throughout.

**Step 4** (not authorized): Run and debug test scripts once documentation makes sense.

**Step 5** (not authorized): Summarize and publish results and source code.

### Test Data

Test data is large (~2.4GB) and not stored in repo. Location controlled by:

1. `LFS_TEST_DATA` environment variable
2. Config file `test_data_path` setting
3. Standard locations (see `pkg/testdata/generator.go`)

Data includes: Big Buck Bunny videos (CC BY 3.0), Project Gutenberg archives, NYC taxi datasets, test PDFs.
