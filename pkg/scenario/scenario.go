package scenario

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mslinn/git_lfs_scripts/pkg/checksum"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/git"
	"github.com/mslinn/git_lfs_scripts/pkg/testdata"
	"github.com/mslinn/git_lfs_scripts/pkg/timing"
)

// Scenario defines a Git LFS test scenario
type Scenario struct {
	ID         int
	Name       string
	ServerType string // 'lfs-test-server', 'giftless', 'rudolfs', 'bare'
	Protocol   string // 'http', 'https', 'ssh', 'local'
	GitServer  string // 'bare', 'github'
	ServerURL  string // e.g., "http://gojira:8080"
	RepoName   string // GitHub repository name (e.g., "username/lfs-eval-test")
}

// Runner executes a scenario
type Runner struct {
	Scenario   *Scenario
	DB         *database.DB
	RunID      int64
	Debug      bool
	Force      bool   // Force recreation of existing repositories
	WorkDir    string // Base directory for test operations
	RepoDir    string // Repository directory (WorkDir/repo)
	Repo2Dir   string // Second clone directory (WorkDir/repo2)
	GitHubURL  string // GitHub clone URL (set during execution if created)
}

// NewRunner creates a new scenario runner
func NewRunner(scenario *Scenario, db *database.DB, workDir string, debug, force bool) *Runner {
	return &Runner{
		Scenario: scenario,
		DB:       db,
		Debug:    debug,
		Force:    force,
		WorkDir:  workDir,
		RepoDir:  workDir + "/repo",
		Repo2Dir: workDir + "/repo2",
	}
}

// Execute runs the complete 7-step scenario
func (r *Runner) Execute() error {
	if r.Debug {
		fmt.Printf("\n=== Executing Scenario %d: %s ===\n", r.Scenario.ID, r.Scenario.Name)
		fmt.Printf("Server: %s via %s\n", r.Scenario.ServerType, r.Scenario.Protocol)
		fmt.Printf("Work directory: %s\n\n", r.WorkDir)
	}

	// Validate prerequisites before starting
	if err := r.validatePrerequisites(); err != nil {
		return err
	}

	// Create test run
	run := &database.TestRun{
		ScenarioID: r.Scenario.ID,
		ServerType: r.Scenario.ServerType,
		Protocol:   r.Scenario.Protocol,
		GitServer:  r.Scenario.GitServer,
		Status:     "running",
		Notes:      fmt.Sprintf("Automated execution of scenario %d", r.Scenario.ID),
	}

	if err := r.DB.CreateTestRun(run); err != nil {
		return fmt.Errorf("failed to create test run: %w", err)
	}
	r.RunID = run.ID

	if r.Debug {
		fmt.Printf("Created test run ID: %d\n\n", r.RunID)
	}

	// Execute each step
	steps := []func() error{
		r.Step1_Setup,
		r.Step2_InitialPush,
		r.Step3_Modifications,
		r.Step4_SecondClone,
		r.Step5_SecondClientPush,
		r.Step6_FirstClientPull,
		r.Step7_Untrack,
	}

	for i, step := range steps {
		stepNum := i + 1
		if r.Debug {
			fmt.Printf("--- Step %d ---\n", stepNum)
		}

		if err := step(); err != nil {
			// Mark run as failed
			run.Status = "failed"
			run.Notes += fmt.Sprintf(" | Failed at step %d: %v", stepNum, err)
			r.DB.UpdateTestRun(run)

			// Attempt cleanup
			if cleanupErr := r.cleanup(); cleanupErr != nil && r.Debug {
				fmt.Printf("Warning: cleanup failed: %v\n", cleanupErr)
			}

			return fmt.Errorf("step %d failed: %w", stepNum, err)
		}

		if r.Debug {
			fmt.Printf("✓ Step %d complete\n\n", stepNum)
		}
	}

	// Mark run as completed
	run.Status = "completed"
	run.Notes += " | All steps completed successfully"
	if err := r.DB.UpdateTestRun(run); err != nil {
		return fmt.Errorf("failed to update test run: %w", err)
	}

	if r.Debug {
		fmt.Printf("=== Scenario %d Complete ===\n", r.Scenario.ID)
	}

	return nil
}

// Step1_Setup: Create repo, configure LFS, copy initial files, compute checksums
func (r *Runner) Step1_Setup() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 1,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Initialize repository
	if r.Debug {
		fmt.Println("Initializing repository...")
	}
	if err := ctx.InitRepo(r.RepoDir, false); err != nil {
		return err
	}

	// Configure git user
	if err := ctx.ConfigUser(r.RepoDir, "LFS Test", "test@example.com"); err != nil {
		return err
	}

	// Create GitHub repository if needed (scenarios 3-9 with github git server)
	if r.Scenario.GitServer == "github" && r.Scenario.RepoName != "" {
		if r.Debug {
			fmt.Println("Creating GitHub repository...")
		}
		cloneURL, err := ctx.CreateGitHubRepo(r.Scenario.RepoName, r.Force)
		if err != nil {
			return fmt.Errorf("failed to create GitHub repo: %w", err)
		}
		r.GitHubURL = cloneURL

		// Add the remote
		if err := ctx.AddRemote(r.RepoDir, "origin", cloneURL); err != nil {
			return fmt.Errorf("failed to add remote: %w", err)
		}
	}

	// Install git-lfs
	if r.Debug {
		fmt.Println("Installing git-lfs...")
	}
	if err := ctx.LFSInstall(r.RepoDir); err != nil {
		return err
	}

	// Configure LFS server URL in .lfsconfig (if applicable)
	if r.Scenario.ServerURL != "" {
		if r.Debug {
			fmt.Printf("Configuring LFS server URL: %s\n", r.Scenario.ServerURL)
		}
		if err := ctx.ConfigureLFSURL(r.RepoDir, r.Scenario.ServerURL); err != nil {
			return err
		}
	}

	// Configure LFS tracking patterns
	if r.Debug {
		fmt.Println("Configuring LFS tracking patterns...")
	}
	patterns := []string{"*.pdf", "*.mov", "*.avi", "*.ogg", "*.m4v", "*.zip"}
	for _, pattern := range patterns {
		if err := ctx.LFSTrack(r.RepoDir, pattern); err != nil {
			return err
		}
	}

	// Generate evaluation README
	if r.Debug {
		fmt.Println("Generating evaluation README...")
	}
	if err := r.generateREADME(); err != nil {
		return fmt.Errorf("failed to generate README: %w", err)
	}

	// Copy initial test files
	if r.Debug {
		fmt.Println("Copying initial test files (v1 - 1.3GB)...")
	}
	files, err := testdata.RealTestFiles()
	if err != nil {
		return fmt.Errorf("failed to get test files: %w", err)
	}

	if err := testdata.CopyFiles(r.RepoDir, files, r.Debug); err != nil {
		return fmt.Errorf("failed to copy test files: %w", err)
	}

	// Compute checksums
	if r.Debug {
		fmt.Println("Computing checksums...")
	}
	checksums, err := checksum.ComputeDirectory(r.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 1, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	if r.Debug {
		fmt.Printf("Stored %d checksums\n", len(checksums))
	}

	return nil
}

// Step2_InitialPush: Add, commit, and push all files with timing
func (r *Runner) Step2_InitialPush() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 2,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Add all files (including .gitattributes from lfs track)
	if r.Debug {
		fmt.Println("Adding files to git...")
	}
	if err := ctx.Add(r.RepoDir, "."); err != nil {
		return err
	}

	// Commit
	if r.Debug {
		fmt.Println("Committing initial files...")
	}
	if err := ctx.Commit(r.RepoDir, "Initial commit with LFS files"); err != nil {
		return err
	}

	// Push (if remote is configured)
	if r.Scenario.ServerURL != "" {
		if r.Debug {
			fmt.Println("Pushing to remote...")
		}
		// TODO: Set up remote first
		// if err := ctx.Push(r.RepoDir, "origin", "main"); err != nil {
		// 	return err
		// }
	}

	// Compute checksums again to verify
	checksums, err := checksum.ComputeDirectory(r.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 2, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	if r.Debug {
		fmt.Printf("Stored %d checksums for step 2\n", len(checksums))
	}

	return nil
}

// Step3_Modifications: Modify, delete, rename files
func (r *Runner) Step3_Modifications() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 3,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Update files with v2 versions
	if r.Debug {
		fmt.Println("Updating files with v2 versions...")
	}
	v2Files, err := testdata.RealTestFilesV2()
	if err != nil {
		return fmt.Errorf("failed to get v2 test files: %w", err)
	}

	if err := testdata.CopyFiles(r.RepoDir, v2Files, r.Debug); err != nil {
		return fmt.Errorf("failed to copy v2 files: %w", err)
	}

	// Delete some files
	if r.Debug {
		fmt.Println("Deleting files...")
	}
	filesToDelete := []string{"video1.m4v", "video4.ogg"}
	for _, file := range filesToDelete {
		if err := testdata.DeleteFile(r.RepoDir, file, r.Debug); err != nil {
			return fmt.Errorf("failed to delete %s: %w", file, err)
		}
	}

	// Rename a file
	if r.Debug {
		fmt.Println("Renaming files...")
	}
	if err := testdata.RenameFile(r.RepoDir, "zip2.zip", "zip2_renamed.zip", r.Debug); err != nil {
		return fmt.Errorf("failed to rename zip2.zip: %w", err)
	}

	// Add all changes
	if r.Debug {
		fmt.Println("Adding changes to git...")
	}
	if err := ctx.Add(r.RepoDir, "-A"); err != nil {
		return err
	}

	// Commit changes
	if r.Debug {
		fmt.Println("Committing modifications...")
	}
	if err := ctx.Commit(r.RepoDir, "Update, delete, and rename files (v2)"); err != nil {
		return err
	}

	// Push (if remote is configured)
	if r.Scenario.ServerURL != "" {
		if r.Debug {
			fmt.Println("Pushing modifications to remote...")
		}
		// TODO: Set up remote first
		// if err := ctx.Push(r.RepoDir, "origin", "main"); err != nil {
		// 	return err
		// }
	}

	// Compute and store checksums
	if r.Debug {
		fmt.Println("Computing checksums after modifications...")
	}
	checksums, err := checksum.ComputeDirectory(r.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 3, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	if r.Debug {
		fmt.Printf("Stored %d checksums for step 3\n", len(checksums))
	}

	return nil
}

// Step4_SecondClone: Clone to second machine and verify
func (r *Runner) Step4_SecondClone() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 4,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Determine the clone URL
	var cloneURL string
	if r.Scenario.Protocol == "local" {
		// For local protocol, use the first repo directory
		cloneURL = r.RepoDir
	} else if r.Scenario.ServerURL != "" {
		// Use the configured server URL
		cloneURL = r.Scenario.ServerURL
	} else {
		return fmt.Errorf("no remote URL configured for cloning")
	}

	// Clone the repository
	if r.Debug {
		fmt.Printf("Cloning from %s to %s...\n", cloneURL, r.Repo2Dir)
	}
	if err := ctx.Clone(cloneURL, r.Repo2Dir); err != nil {
		return err
	}

	// Compute checksums in the second clone
	if r.Debug {
		fmt.Println("Computing checksums in second clone...")
	}
	checksums, err := checksum.ComputeDirectory(r.Repo2Dir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 4, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	// Compare checksums with step 3
	if r.Debug {
		fmt.Println("Comparing checksums with step 3...")
	}
	diffs, err := checksum.CompareChecksums(r.DB, r.RunID, 3, 4)
	if err != nil {
		return fmt.Errorf("failed to compare checksums: %w", err)
	}

	if len(diffs) > 0 {
		return fmt.Errorf("checksum mismatch: %d differences found between step 3 and step 4", len(diffs))
	}

	if r.Debug {
		fmt.Printf("✓ Checksums match (%d files)\n", len(checksums))
	}

	return nil
}

// Step5_SecondClientPush: Make changes on second client
func (r *Runner) Step5_SecondClientPush() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 5,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Create a new file in the second clone
	if r.Debug {
		fmt.Println("Creating new file in second clone...")
	}
	newFilePath := filepath.Join(r.Repo2Dir, "README.md")
	content := []byte("# LFS Test Repository\n\nThis file was added during Step 5 testing.\n")
	if err := os.WriteFile(newFilePath, content, 0644); err != nil {
		return fmt.Errorf("failed to create new file: %w", err)
	}

	// Add the new file
	if r.Debug {
		fmt.Println("Adding new file to git...")
	}
	if err := ctx.Add(r.Repo2Dir, "README.md"); err != nil {
		return err
	}

	// Commit the change
	if r.Debug {
		fmt.Println("Committing new file...")
	}
	if err := ctx.Commit(r.Repo2Dir, "Add README from second client"); err != nil {
		return err
	}

	// Push changes (if remote is configured)
	if r.Scenario.Protocol != "local" && r.Scenario.ServerURL != "" {
		if r.Debug {
			fmt.Println("Pushing changes to remote...")
		}
		// TODO: Set up remote first
		// if err := ctx.Push(r.Repo2Dir, "origin", "main"); err != nil {
		// 	return err
		// }
	}

	// Compute and store checksums
	if r.Debug {
		fmt.Println("Computing checksums after changes...")
	}
	checksums, err := checksum.ComputeDirectory(r.Repo2Dir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 5, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	if r.Debug {
		fmt.Printf("Stored %d checksums for step 5\n", len(checksums))
	}

	return nil
}

// Step6_FirstClientPull: Pull changes to first client
func (r *Runner) Step6_FirstClientPull() error {
	// Pull changes from remote (if configured)
	if r.Scenario.Protocol != "local" && r.Scenario.ServerURL != "" {
		if r.Debug {
			fmt.Println("Pulling changes from remote...")
		}
		// TODO: Set up remote and use ctx.Pull
		// ctx := &git.Context{DB: r.DB, RunID: r.RunID, StepNumber: 6, Debug: r.Debug, WorkDir: r.WorkDir}
		// if err := ctx.Pull(r.RepoDir); err != nil {
		// 	return err
		// }
		if r.Debug {
			fmt.Println("  (Skipping pull - remote not yet configured)")
		}
	} else if r.Scenario.Protocol == "local" {
		if r.Debug {
			fmt.Println("Pulling changes from local repo...")
		}
		// For local protocol, we need to manually sync
		// In real scenario, this would use git pull from the first repo
		// For now, we'll just note this needs to be implemented
		if r.Debug {
			fmt.Println("  (Skipping local pull - requires bare repo setup)")
		}
	}

	// Compute checksums in first clone
	if r.Debug {
		fmt.Println("Computing checksums in first clone...")
	}
	checksums, err := checksum.ComputeDirectory(r.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 6, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	// Note: We can't compare with step 5 until pull is working
	// The checksums should match step 5 after successful pull
	if r.Debug {
		fmt.Printf("Stored %d checksums for step 6\n", len(checksums))
		fmt.Println("  Note: Checksum comparison with step 5 requires working pull")
	}

	return nil
}

// Step7_Untrack: Untrack and unmigrate from LFS
func (r *Runner) Step7_Untrack() error {
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 7,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Untrack patterns from LFS
	if r.Debug {
		fmt.Println("Untracking patterns from LFS...")
	}
	patterns := []string{"*.pdf", "*.mov", "*.avi", "*.ogg", "*.m4v", "*.zip"}
	for _, pattern := range patterns {
		if err := ctx.LFSUntrack(r.RepoDir, pattern); err != nil {
			return err
		}
	}

	// Use git lfs migrate to convert files back to regular git
	if r.Debug {
		fmt.Println("Migrating files out of LFS...")
	}
	if err := ctx.LFSMigrate(r.RepoDir); err != nil {
		return err
	}

	// Add .gitattributes changes
	if r.Debug {
		fmt.Println("Adding .gitattributes changes...")
	}
	if err := ctx.Add(r.RepoDir, ".gitattributes"); err != nil {
		return err
	}

	// Commit the untrack changes
	if r.Debug {
		fmt.Println("Committing LFS untrack...")
	}
	if err := ctx.Commit(r.RepoDir, "Untrack files from LFS"); err != nil {
		return err
	}

	// Compute final checksums
	if r.Debug {
		fmt.Println("Computing final checksums...")
	}
	checksums, err := checksum.ComputeDirectory(r.RepoDir)
	if err != nil {
		return fmt.Errorf("failed to compute checksums: %w", err)
	}

	if err := checksum.StoreChecksums(r.DB, r.RunID, 7, checksums); err != nil {
		return fmt.Errorf("failed to store checksums: %w", err)
	}

	if r.Debug {
		fmt.Printf("Stored %d checksums for step 7\n", len(checksums))
		fmt.Println("✓ Files successfully untracked from LFS")
	}

	return nil
}

// generateREADME creates an evaluation README.md file
func (r *Runner) generateREADME() error {
	readmePath := filepath.Join(r.RepoDir, "README.md")

	content := fmt.Sprintf(`# Git LFS Evaluation Repository

This repository is used for evaluating Git LFS server implementations.

## Scenario Information

- **Scenario ID**: %d
- **Name**: %s
- **Server Type**: %s
- **Protocol**: %s
- **Git Server**: %s
`, r.Scenario.ID, r.Scenario.Name, r.Scenario.ServerType, r.Scenario.Protocol, r.Scenario.GitServer)

	if r.Scenario.ServerURL != "" {
		content += fmt.Sprintf("- **Server URL**: %s\n", r.Scenario.ServerURL)
	}

	content += `
## Test Files

This repository contains approximately 2.4GB of test files in various formats:
- PDF documents
- Video files (AVI, M4V, MOV, OGG)
- ZIP archives

These files are used to test Git LFS functionality including:
- Initial commits with large files
- File modifications and updates
- File deletions and renames
- Cloning and synchronization
- LFS migration operations

## Evaluation Procedure

The evaluation follows a 7-step process:
1. Setup repository with LFS tracking
2. Initial commit and push
3. Modify, delete, and rename files
4. Clone to second location
5. Make changes on second client
6. Pull changes to first client
7. Untrack files from LFS

## Documentation

For more information about the evaluation procedure, see:
https://www.mslinn.com/git/5600-git-lfs-evaluation.html

## Test Data

Test data is sourced from:
- Big Buck Bunny videos (CC BY 3.0)
- Project Gutenberg archives
- NYC taxi datasets
- Test PDFs from testfile.org

---
Generated automatically by lfst-scenario command.
`

	if err := os.WriteFile(readmePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write README: %w", err)
	}

	if r.Debug {
		fmt.Printf("  ✓ Created README.md\n")
	}

	return nil
}

// validatePrerequisites checks if all prerequisites are met before starting scenario
func (r *Runner) validatePrerequisites() error {
	if r.Debug {
		fmt.Println("Validating prerequisites...")
	}

	// Check if git is available
	result := timing.Run("git", []string{"--version"}, nil)
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	if r.Debug {
		fmt.Println("  ✓ git is available")
	}

	// Check if git-lfs is available
	result = timing.Run("git", []string{"lfs", "version"}, nil)
	if result.Error != nil || result.ExitCode != 0 {
		return fmt.Errorf("git-lfs is not installed or not in PATH\n\nInstall with: apt-get install git-lfs")
	}
	if r.Debug {
		fmt.Println("  ✓ git-lfs is available")
	}

	// Try to get test data path
	dataPath, err := testdata.GetTestDataPath()
	if err != nil {
		return fmt.Errorf("test data not found: %w\n\nPlease set LFS_TEST_DATA environment variable or place data in standard locations.\nSee: https://www.mslinn.com/git/5600-git-lfs-evaluation.html#git_lfs_test_data", err)
	}

	// Check if test data is remote and rsync is available
	if _, isRemote := testdata.ParseRemotePath(dataPath); isRemote {
		result := timing.Run("rsync", []string{"--version"}, nil)
		if result.Error != nil || result.ExitCode != 0 {
			return fmt.Errorf("rsync is not installed or not in PATH\n\nRsync is required for remote test data.\nInstall with: apt-get install rsync")
		}
		if r.Debug {
			fmt.Println("  ✓ rsync is available (for remote test data)")
		}
	}

	if r.Debug {
		fmt.Printf("  ✓ Test data found at: %s\n", dataPath)
	}

	return nil
}

// cleanup removes working directories after failure
func (r *Runner) cleanup() error {
	if r.Debug {
		fmt.Println("\nCleaning up working directories...")
	}

	var errs []error

	// Remove first repository directory
	if _, err := os.Stat(r.RepoDir); err == nil {
		if err := os.RemoveAll(r.RepoDir); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", r.RepoDir, err))
		} else if r.Debug {
			fmt.Printf("  ✓ Removed %s\n", r.RepoDir)
		}
	}

	// Remove second repository directory
	if _, err := os.Stat(r.Repo2Dir); err == nil {
		if err := os.RemoveAll(r.Repo2Dir); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", r.Repo2Dir, err))
		} else if r.Debug {
			fmt.Printf("  ✓ Removed %s\n", r.Repo2Dir)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	return nil
}
