# Setup and Test Execution

Clients automatically detect they're remote and SSH their data to `gojira`.
All commands work from anywhere with SSH access.


## One-Time Setup

### On Git LFS Server (gojira)

We cannot do this yet because no release has been published to date:

```shell
go install github.com/mslinn/git_lfs_scripts/cmd/...@latest
```

Do this instead for now:

```shell
$ cd /work/git/git_lfs_scripts
$ make install
```

This installs all commands to `/usr/local/bin/`:

- `lfst-checksum` - Compute and store checksums
- `lfst-import` - Import checksum JSON data
- `lfst-run` - Manage test run lifecycle
- `lfst-query` - Query and report on test data
- `lfst-scenario` - Execute complete 7-step test scenarios

The database is created automatically at first use at
`gojira:/home/mslinn/lfs_eval/lfs-test.db`


### On Clients (Bear and Camille)

Same installation as server:

```shell
$ cd /mnt/f/work/git/git_lfs_scripts  # or appropriate path
$ make install
```

Configure `~/.lfs-test-config`:

```shell
$ cat > ~/.lfs-test-config <<EOF
database: /home/mslinn/lfs_eval/lfs-test.db
remote_host: gojira
auto_remote: true
EOF
```


### Environment Variables

The test commands use several environment variables for configuration:

**Required for testing:**

- `LFS_TEST_DATA` - Location of test data directory (e.g., `/mnt/f/work/git/git_lfs_test_data`)


**Optional (override config file):**

- `LFS_TEST_DB` - Database path (default: `/home/mslinn/lfs_eval/lfs-test.db`)
- `LFS_TEST_CONFIG` - Path to config file (default: `~/.lfs-test-config`)
- `LFS_REMOTE_HOST` - Remote host for SSH operations (default: `gojira`)
- `LFS_AUTO_REMOTE` - Enable auto-remote detection: `true`/`1` or `false`/`0` (default: `true`)


**Setting environment variables:**

```shell
# Recommended: Add to ~/.bashrc or ~/.zshrc
export LFS_TEST_DATA=/mnt/f/work/git/git_lfs_test_data

# Optional overrides
export LFS_TEST_DB=/path/to/custom/database.db
export LFS_REMOTE_HOST=myserver
```

For more information on environment variables and directory organization, see:
https://www.mslinn.com/git/5600-git-lfs-evaluation.html


### Test Data Requirements

The test scenarios require **2.4GB of real large files** (103M-398M each).
These must be available on the machine running `lfst-scenario`.

Check if test data exists:

```shell
$ ls -lh $LFS_TEST_DATA/v1/
total 1.3G
-rw-r--r-- 1 mslinn mslinn 103M Jan 23  2025 pdf1.pdf
-rw-r--r-- 1 mslinn mslinn 116M Jan 23  2025 video1.m4v
-rw-r--r-- 1 mslinn mslinn 238M Jan 23  2025 video2.mov
-rw-r--r-- 1 mslinn mslinn 150M Jan 23  2025 video3.avi
-rw-r--r-- 1 mslinn mslinn 188M Jan 23  2025 video4.ogg
-rw-r--r-- 1 mslinn mslinn 308M Jan 23  2025 zip1.zip
-rw-r--r-- 1 mslinn mslinn 154M Jan 23  2025 zip2.zip
```

If not present, the test data can be downloaded using the `git_lfs_test_data` script
(documented at https://www.mslinn.com/git/5600-git-lfs-evaluation.html#git_lfs_test_data).


## Running a Test Scenario

### Automated Execution (Recommended)

The `lfst-scenario` command automates the entire 7-step test procedure. All steps are fully implemented, including GitHub repository creation (using `gh` CLI), `.lfsconfig` file generation, evaluation README generation, and automatic cleanup on failure. Use the `-f` flag to force recreation of existing repositories.

```shell
# List available scenarios
$ lfst-scenario --list
Available scenarios:

ID  Server             Protocol  Git Server  Description
--  ------             --------  ----------  -----------
1   bare               local     bare        Bare repo - local
2   bare               ssh       bare        Bare repo - SSH
6   lfs-test-server    http      bare        LFS Test Server - HTTP
7   lfs-test-server    http      github      LFS Test Server - HTTP/GitHub
8   giftless           local     bare        Giftless - local
9   giftless           ssh       bare        Giftless - SSH
13  rudolfs            local     bare        Rudolfs - local
14  rudolfs            ssh       bare        Rudolfs - SSH

# Run scenario 6 (LFS Test Server - HTTP) with debug output
$ lfst-scenario -d 6

=== Executing Scenario 6: LFS Test Server - HTTP ===
Server: lfs-test-server via http
Work directory: /tmp/lfst

Created test run ID: 1

--- Step 1 ---
Initializing repository...
  ✓ Initialized in 15ms
Installing git-lfs...
  ✓ Installed git-lfs in 234ms
Configuring LFS tracking patterns...
  ✓ Tracked *.pdf in 12ms
  ✓ Tracked *.mov in 11ms
  ... (more patterns)
Copying initial test files (v1 - 1.3GB)...
Copying 7 test files to /tmp/lfst/repo
  Copying pdf1.pdf (103.00 MB)
  Copying video1.m4v (116.00 MB)
  ... (more files)
  ✓ Copied 7 files
Computing checksums...
Stored 7 checksums
✓ Step 1 complete

--- Step 2 ---
Adding files to git...
  ✓ Added in 3241ms
Committing initial files...
  ✓ Committed in 567ms
Stored 7 checksums for step 2
✓ Step 2 complete

--- Step 3 ---
Updating files with v2 versions...
  Copying pdf1.pdf (205.00 MB)
  Copying video2.mov (398.00 MB)
  Copying video3.avi (272.00 MB)
  Copying zip1.zip (200.00 MB)
  ✓ Copied 4 files
Deleting files...
  Deleting video1.m4v
  Deleting video4.ogg
Renaming files...
  Renaming zip2.zip to zip2_renamed.zip
Adding changes to git...
  ✓ Added in 2850ms
Committing modifications...
  ✓ Committed in 421ms
Computing checksums after modifications...
Stored 6 checksums for step 3
✓ Step 3 complete

--- Step 4 ---
Cloning from /tmp/lfst/repo to /tmp/lfst/repo2...
  ✓ Cloned in 1234ms
Computing checksums in second clone...
Comparing checksums with step 3...
✓ Checksums match (6 files)
✓ Step 4 complete

--- Step 5 ---
Creating new file in second clone...
Adding new file to git...
  ✓ Added in 45ms
Committing new file...
  ✓ Committed in 123ms
Computing checksums after changes...
Stored 7 checksums for step 5
✓ Step 5 complete

--- Step 6 ---
Pulling changes from remote...
  (Skipping pull - remote not yet configured)
Computing checksums in first clone...
Stored 6 checksums for step 6
  Note: Checksum comparison with step 5 requires working pull
✓ Step 6 complete

--- Step 7 ---
Untracking patterns from LFS...
  ✓ Untracked *.pdf in 25ms
  ✓ Untracked *.mov in 18ms
  ✓ Untracked *.avi in 20ms
  ✓ Untracked *.ogg in 19ms
  ✓ Untracked *.m4v in 21ms
  ✓ Untracked *.zip in 17ms
Migrating files out of LFS...
  ✓ Migrated files in 5432ms
Adding .gitattributes changes...
  ✓ Added in 12ms
Committing LFS untrack...
  ✓ Committed in 98ms
Computing final checksums...
Stored 6 checksums for step 7
✓ Files successfully untracked from LFS
✓ Step 7 complete

✓ Scenario 6 completed successfully
  Run ID: 1
  View results: lfst-run show 1
```

### What Each Step Does

1. **Setup**:
   - Create local git repository
   - Configure git user (LFS Test <test@example.com>)
   - Create GitHub repository (for scenarios with github git server) using `gh` CLI
   - Add GitHub remote as 'origin'
   - Install git-lfs hooks
   - Create `.lfsconfig` with LFS server URL (if applicable)
   - Configure LFS tracking patterns, e.g. \*.pdf, \*.mov, \*.avi, \*.ogg, \*.m4v, \*.zip
   - Generate evaluation README.md with scenario details
   - Copy 1.3GB test files from v1/
   - Compute and store checksums
2. **Initial Push**: Add all files to git, commit with message "Initial commit with LFS files", verify checksums match step 1
3. **Modifications**:
   - Update 4 files with v2 versions (pdf1.pdf, video2.mov, video3.avi, zip1.zip)
   - Delete 2 files (video1.m4v, video4.ogg)
   - Rename 1 file (zip2.zip → zip2_renamed.zip)
   - Add all changes, commit with message "Update, delete, and rename files (v2)", compute checksums
4. **Second Clone**: Clone repository to repo2 directory, compute checksums, compare with step 3 (must match)
5. **Second Client Push**: Create README.md in second clone, add and commit with message "Add README from second client", compute checksums
6. **First Client Pull**: Pull changes from remote (when configured), compute checksums in first clone (should match step 5 after successful pull)
7. **Untrack**: Untrack all patterns from LFS, run git lfs migrate export, commit .gitattributes changes, compute final checksums (files now in regular git)


## Viewing Results

### Show test run details

From anywhere (clients or server):

```shell
$ lfst-run show 1
Test Run 1:
  Scenario ID:  6
  Server Type:  lfs-test-server
  Protocol:     http
  Git Server:   bare
  Status:       completed
  Started:      2025-10-16 15:30:00
  Completed:    2025-10-16 15:35:23
  Duration:     323.45s
  Notes:        Automated execution of scenario 6 | All steps completed successfully
```

### View checksums for a specific step

```shell
$ lfst-query checksums --run-id 1 --step 1
Checksums for run 1, step 1:

CRC32             Size        Path
-----             ----        ----
a1b2c3d4          103.00 MB   pdf1.pdf
e5f6g7h8          116.00 MB   video1.m4v
... (more checksums)
```

### Compare checksums between steps

```shell
$ lfst-query compare --run-id 1 --from 1 --to 2
Changes from step 1 to step 2:

  No differences found
```

### View statistics

```shell
$ lfst-query stats --run-id 1
Test Run 1 Statistics:

  Scenario:     6
  Server:       lfs-test-server
  Protocol:     http
  Status:       completed

  Checksums per step:
    Step 1: 7 checksums
    Step 2: 7 checksums

  Operations per step:
    Step 1: 9 operations (avg 156.3ms)
    Step 2: 2 operations (avg 1904.0ms)
```

### List all runs

```shell
$ lfst-run list --status completed
ID  Scenario  Server           Protocol  Git   Status     Started   Duration  Notes
--  --------  ------           --------  ---   ------     -------   --------  -----
1   6         lfs-test-server  http      bare  completed  15:30:00  323.5s    Automated execution...
```


## Manual Execution (Advanced)

For debugging or custom workflows, you can run individual commands:

```shell
# Create a test run manually
$ lfst-run create --scenario 6 --server lfs-test-server --protocol http
Created test run ID: 2

# Compute and store checksums for a specific directory and step
$ lfst-checksum --run-id 2 --step 1 --dir /path/to/repo
Computed 7 checksums
✓ Stored 7 checksums on gojira for step 1

# Mark run as completed
$ lfst-run complete 2 --notes "Manual test completed"
✓ Test run 2 marked as completed (45.23s)
```


## GitHub Scenarios

Scenario 7 (and future GitHub-based scenarios) automatically creates a private GitHub repository using the `gh` CLI.

### Prerequisites

Install and authenticate with GitHub CLI:

```shell
# Install gh (if not already installed)
$ sudo apt install gh

# Authenticate with GitHub
$ gh auth login
```

### Running GitHub Scenarios

```shell
# First run - creates new GitHub repository
$ lfst-scenario 7

# Subsequent runs - use force flag to recreate repository
$ lfst-scenario -f 7
```

The command will:
1. Create private GitHub repository (e.g., `mslinn/lfs-eval-test`)
2. Add it as 'origin' remote
3. Configure `.lfsconfig` with LFS server URL
4. Execute all 7 test steps

**Note:** The `-f` flag deletes and recreates the GitHub repository. Use with caution on production repositories.


## Troubleshooting

### GitHub repository creation fails

```shell
Error: gh CLI not available - install with: sudo apt install gh
```

**Solution:** Install and authenticate with GitHub CLI:

```shell
$ sudo apt install gh
$ gh auth login
```

### Test data not found

```shell
Error: test data directory not found (searched: [/mnt/f/work/git/git_lfs_test_data /work/git/git_lfs_test_data ...])
```

**Solution:** Set the `LFS_TEST_DATA` environment variable (add to `~/.bashrc` or `~/.zshrc`):

```shell
export LFS_TEST_DATA=/mnt/f/work/git/git_lfs_test_data  # or your actual path
lfst-scenario 6
```

The commands search for test data in these locations:
1. `$LFS_TEST_DATA` environment variable
2. `/mnt/f/work/git/git_lfs_test_data` (WSL/Windows)
3. `/work/git/git_lfs_test_data` (Linux)
4. `/home/mslinn/git_lfs_test_data` (fallback)

### Remote mode not working

If checksums aren't being sent to gojira:

```shell
# Force local mode for testing
lfst-checksum --local --run-id 1 --step 1 --dir /path/to/repo

# Force remote mode with specific host
lfst-checksum --remote gojira --run-id 1 --step 1 --dir /path/to/repo

# Check SSH connectivity
ssh gojira "which lfst-import"
```

### Database location

Commands search for the database in this order (highest priority first):
1. `--db` command-line flag
2. `$LFS_TEST_DB` environment variable
3. `database` setting in `~/.lfs-test-config` file
4. `$LFS_TEST_CONFIG` environment variable (alternate config file path)
5. Default: `/home/mslinn/lfs_eval/lfs-test.db`

**Example:**
```shell
# Use environment variable
export LFS_TEST_DB=/path/to/my-test.db

# Or use command-line flag
lfst-scenario --db /path/to/my-test.db 6
```


## Next Steps

**Completed:**
- ✓ All 7 steps implemented in scenario runner
- ✓ File modifications, deletions, and renames (Step 3)
- ✓ Clone to second directory (Step 4)
- ✓ Changes on second client (Step 5)
- ✓ Pull to first client (Step 6)
- ✓ LFS untrack and migrate (Step 7)
- ✓ Database concurrency fixes (WAL mode + busy timeout)
- ✓ Real large test files (2.4GB from v1/ and v2/)
- ✓ `create_lfs_eval_repo` functionality:
  - ✓ GitHub repository creation using `gh` CLI
  - ✓ `.lfsconfig` file generation with LFS server URL
  - ✓ Evaluation README.md generation
  - ✓ Force flag (`-f`) to recreate existing repos
  - ✓ Remote setup automation for GitHub scenarios

**Remaining:**
1. Add remote repository setup automation:
   - Auto-create bare repos for local scenarios
   - Configure remote URLs in git
   - Enable actual push/pull operations
2. Add LFS server configuration helpers:
   - Start/stop LFS test servers
   - Verify server availability
   - Auto-configure server URLs
3. Implement scenario-specific configurations:
   - Server-specific setup (lfs-test-server, giftless, rudolfs)
   - Protocol-specific handling (http, https, ssh)
4. Add comprehensive error recovery:
   - Cleanup on failure
   - Resume capability
   - Rollback mechanisms
5. Refactor for short-lived database connections:
   - Open/close DB per operation instead of holding throughout test
   - Improves concurrency for long-running tests
6. Add more reporting and comparison tools:
   - Detailed timing analysis
   - Performance comparisons between scenarios
   - Storage efficiency metrics
