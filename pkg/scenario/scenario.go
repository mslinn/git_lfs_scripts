package scenario

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lithammer/dedent"
	"github.com/mslinn/git_lfs_scripts/pkg/checksum"
	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/git"
	"github.com/mslinn/git_lfs_scripts/pkg/report"
	"github.com/mslinn/git_lfs_scripts/pkg/serverinfo"
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

// GetScenarios returns the predefined scenarios map based on gitScenarios.html
func GetScenarios() map[int]*Scenario {
	return map[int]*Scenario{
		1:  {ID: 1, Name: "Bare repo - local", ServerType: "bare", Protocol: "local", GitServer: "bare"},
		2:  {ID: 2, Name: "Bare repo - SSH", ServerType: "bare", Protocol: "ssh", GitServer: "bare"},
		6:  {ID: 6, Name: "LFS Test Server - HTTP", ServerType: "lfs-test-server", Protocol: "http", GitServer: "bare", ServerURL: "http://gojira:8080"},
		7:  {ID: 7, Name: "LFS Test Server - HTTP/GitHub", ServerType: "lfs-test-server", Protocol: "http", GitServer: "github", ServerURL: "http://gojira:8080", RepoName: "mslinn/lfs-eval-test"},
		8:  {ID: 8, Name: "Giftless - local", ServerType: "giftless", Protocol: "local", GitServer: "bare"},
		9:  {ID: 9, Name: "Giftless - SSH", ServerType: "giftless", Protocol: "ssh", GitServer: "bare"},
		13: {ID: 13, Name: "Rudolfs - local", ServerType: "rudolfs", Protocol: "local", GitServer: "bare"},
		14: {ID: 14, Name: "Rudolfs - SSH", ServerType: "rudolfs", Protocol: "ssh", GitServer: "bare"},
	}
}

// GetScenario returns a single scenario by ID, or error if not found
func GetScenario(id int) (*Scenario, error) {
	scenarios := GetScenarios()
	scenario, ok := scenarios[id]
	if !ok {
		return nil, fmt.Errorf("scenario %d not found", id)
	}
	return scenario, nil
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
	BareRepoURL string // Bare repository URL (set during execution if created)
	ReportPath string // Path to save markdown report (optional)
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
		if r.Scenario.ServerURL != "" {
			fmt.Printf("Server URL: %s\n", r.Scenario.ServerURL)
		}
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

	// Collect server information
	if r.Debug {
		fmt.Println("Collecting server information...")
	}
	collected, err := serverinfo.CollectServerInfo(r.Scenario.ServerURL, r.RunID, r.Debug)
	if err != nil {
		if r.Debug {
			fmt.Printf("Warning: failed to collect server info: %v\n", err)
		}
	} else {
		if err := serverinfo.StoreCollectedInfo(r.DB, collected); err != nil {
			if r.Debug {
				fmt.Printf("Warning: failed to store server info: %v\n", err)
			}
		}
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
		fmt.Printf("=== Scenario %d Complete ===\n\n", r.Scenario.ID)
	}

	// Generate report
	if err := r.generateReport(); err != nil {
		if r.Debug {
			fmt.Printf("Warning: failed to generate report: %v\n", err)
		}
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

	// Create GitHub repository if needed (scenarios with github git server)
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

	// Create bare repository if needed (scenarios with bare git server)
	if r.Scenario.GitServer == "bare" && r.Scenario.ServerURL != "" {
		// Parse the server URL to get the hostname
		parsedURL, err := url.Parse(r.Scenario.ServerURL)
		if err != nil {
			return fmt.Errorf("failed to parse server URL: %w", err)
		}
		hostname := parsedURL.Hostname()

		// Create a unique path for the bare repository on the remote host
		bareRepoPath := fmt.Sprintf("/opt/lfs-test-repos/test-repo-%d.git", r.Scenario.ID)

		if r.Debug {
			fmt.Printf("Creating bare repository on %s...\n", hostname)
		}
		if err := ctx.CreateRemoteBareRepo(hostname, bareRepoPath, r.Force); err != nil {
			return fmt.Errorf("failed to create bare repository: %w", err)
		}

		// Construct the SSH URL for the bare repository
		bareRepoURL := fmt.Sprintf("%s:%s", hostname, bareRepoPath)
		r.BareRepoURL = bareRepoURL

		// Add the bare repository as a remote
		if err := ctx.AddRemote(r.RepoDir, "origin", bareRepoURL); err != nil {
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
	if r.GitHubURL != "" || r.BareRepoURL != "" {
		if r.Debug {
			fmt.Println("Pushing to remote...")
		}
		// Determine the default branch name (try both master and main)
		branchResult := timing.Run("git", []string{"-C", r.RepoDir, "branch", "--show-current"}, nil)
		branch := "master"
		if branchResult.ExitCode == 0 && branchResult.Stdout != "" {
			branch = strings.TrimSpace(branchResult.Stdout)
		}

		if err := ctx.Push(r.RepoDir, "origin", branch); err != nil {
			return err
		}
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
	if r.GitHubURL != "" || r.BareRepoURL != "" {
		if r.Debug {
			fmt.Println("Pushing modifications to remote...")
		}
		// Determine the default branch name
		branchResult := timing.Run("git", []string{"-C", r.RepoDir, "branch", "--show-current"}, nil)
		branch := "master"
		if branchResult.ExitCode == 0 && branchResult.Stdout != "" {
			branch = strings.TrimSpace(branchResult.Stdout)
		}

		if err := ctx.Push(r.RepoDir, "origin", branch); err != nil {
			return err
		}
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
	} else if r.GitHubURL != "" {
		// Use the GitHub repository URL
		cloneURL = r.GitHubURL
	} else if r.BareRepoURL != "" {
		// Use the bare repository URL
		cloneURL = r.BareRepoURL
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
	if r.GitHubURL != "" || r.BareRepoURL != "" {
		if r.Debug {
			fmt.Println("Pushing changes to remote...")
		}
		// Determine the default branch name
		branchResult := timing.Run("git", []string{"-C", r.Repo2Dir, "branch", "--show-current"}, nil)
		branch := "master"
		if branchResult.ExitCode == 0 && branchResult.Stdout != "" {
			branch = strings.TrimSpace(branchResult.Stdout)
		}

		if err := ctx.Push(r.Repo2Dir, "origin", branch); err != nil {
			return err
		}
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
	ctx := &git.Context{
		DB:         r.DB,
		RunID:      r.RunID,
		StepNumber: 6,
		Debug:      r.Debug,
		WorkDir:    r.WorkDir,
	}

	// Pull changes from remote (if configured)
	if r.GitHubURL != "" || r.BareRepoURL != "" {
		if r.Debug {
			fmt.Println("Pulling changes from remote...")
		}
		if err := ctx.Pull(r.RepoDir); err != nil {
			return err
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

	// Check LFS configuration and server connectivity
	if err := r.checkLFSConfiguration(); err != nil {
		return err
	}

	return nil
}

// checkLFSConfiguration validates LFS setup for the scenario type.
// For scenarios with ServerURL, checks server configuration and connectivity.
// For scenarios without ServerURL, warns about implementation status.
func (r *Runner) checkLFSConfiguration() error {
	if r.Scenario.ServerURL != "" {
		// Validate server-specific configuration before checking connectivity
		if err := r.validateServerConfiguration(); err != nil {
			return err
		}

		// Check server connectivity with short timeout
		if err := checkServerConnectivity(r.Scenario.ServerURL, r.Scenario.ServerType, 2*time.Second, r.Debug); err != nil {
			return err
		}
		if r.Debug {
			fmt.Printf("  ✓ LFS server is reachable at: %s\n", r.Scenario.ServerURL)
		}
	} else {
		// Scenarios without ServerURL (e.g., bare local/SSH) may require additional setup
		if r.Scenario.ServerType == "bare" {
			if r.Debug {
				fmt.Printf("  ⚠ Scenario %d uses bare repo without explicit LFS server URL\n", r.Scenario.ID)
				fmt.Println("    This scenario may require a local bare repository to be set up manually")
			}
			// For now, just warn - these scenarios may not be fully implemented
			return fmt.Errorf("scenario %d (%s) is not yet fully implemented\n\nScenarios with bare repositories require:\n  1. A bare git repository to be created\n  2. LFS storage configuration\n  3. Appropriate git remotes\n\nPlease use scenarios 6-7 (LFS Test Server) which are fully implemented",
				r.Scenario.ID, r.Scenario.Name)
		}
	}

	return nil
}

// validateServerConfiguration validates server-specific requirements for the scenario.
// This checks the server's own configuration, not lfst configuration.
func (r *Runner) validateServerConfiguration() error {
	parsedURL, _ := url.Parse(r.Scenario.ServerURL)
	hostname := parsedURL.Hostname()

	if r.Debug {
		fmt.Printf("  Validating %s configuration on %s...\n", r.Scenario.ServerType, hostname)
	}

	switch r.Scenario.ServerType {
	case "lfs-test-server":
		return validateLFSTestServer(hostname, r.Debug)
	case "giftless":
		return validateGiftless(hostname, r.Debug)
	case "rudolfs":
		return validateRudolfs(hostname, r.Debug)
	default:
		// Unknown server type, skip validation
		if r.Debug {
			fmt.Printf("  ⚠ Unknown server type '%s', skipping configuration validation\n", r.Scenario.ServerType)
		}
	}

	return nil
}

// validateLFSTestServer checks if lfs-test-server is properly configured on the remote host.
func validateLFSTestServer(hostname string, debug bool) error {
	// Check if lfs-test-server binary exists in PATH or standard location
	result := timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "which lfs-test-server || test -x /home/mslinn/go/bin/lfs-test-server"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("lfs-test-server binary not found on %s\n\nPlease ensure:\n  1. lfs-test-server is installed (go install github.com/git-lfs/lfs-test-server@latest)\n  2. Binary is in PATH or at /home/mslinn/go/bin/lfs-test-server\n\nSee: https://github.com/git-lfs/lfs-test-server#installation",
			hostname)
	}

	// Check if LFS_CONTENTPATH env var or data directory exists
	// lfs-test-server uses $LFS_CONTENTPATH or ./lfs-test-server-content by default
	result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "test -d /opt/lfs-test-server || echo 'missing'"}, nil)
	if strings.Contains(result.Stdout, "missing") {
		return fmt.Errorf("lfs-test-server data directory not found on %s\n\nPlease create the data directory:\n  ssh %s\n  sudo mkdir -p /opt/lfs-test-server\n  sudo chown $USER:$USER /opt/lfs-test-server\n\nOr set LFS_CONTENTPATH env var on %s to an existing directory",
			hostname, hostname, hostname)
	}

	// Check if data directory is writable
	result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "test -w /opt/lfs-test-server"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("lfs-test-server data directory /opt/lfs-test-server on %s is not writable\n\nFix permissions:\n  ssh %s\n  sudo chown $USER:$USER /opt/lfs-test-server\n  chmod 755 /opt/lfs-test-server",
			hostname, hostname)
	}

	if debug {
		fmt.Printf("  ✓ lfs-test-server configuration validated\n")
	}

	return nil
}

// validateGiftless checks if giftless is properly configured on the remote host.
func validateGiftless(hostname string, debug bool) error {
	// Check if giftless is installed
	result := timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "which giftless || test -x /work/git/giftless/venv/bin/giftless"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("giftless not found on %s\n\nPlease install giftless:\n  ssh %s\n  cd /work/git\n  git clone https://github.com/datopian/giftless.git\n  cd giftless\n  python3 -m venv venv\n  source venv/bin/activate\n  pip install -e .\n\nSee: https://github.com/datopian/giftless#installation",
			hostname, hostname)
	}

	// Check if giftless config exists
	result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "test -f /work/git/giftless/giftless.yaml"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("giftless configuration file not found on %s\n\nCreate /work/git/giftless/giftless.yaml with storage backend configuration\n\nSee: https://github.com/datopian/giftless#configuration",
			hostname)
	}

	if debug {
		fmt.Printf("  ✓ giftless configuration validated\n")
	}

	return nil
}

// validateRudolfs checks if rudolfs is properly configured on the remote host.
func validateRudolfs(hostname string, debug bool) error {
	// Check if rudolfs binary exists
	result := timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "which rudolfs || test -x /work/git/rudolfs/target/release/rudolfs"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("rudolfs not found on %s\n\nPlease install rudolfs:\n  ssh %s\n  cd /work/git\n  git clone https://github.com/jasonwhite/rudolfs.git\n  cd rudolfs\n  cargo build --release\n\nSee: https://github.com/jasonwhite/rudolfs#building",
			hostname, hostname)
	}

	// Check if rudolfs storage directory exists
	result = timing.Run("ssh", []string{"-o", "ConnectTimeout=5", hostname, "test -d /work/git/rudolfs/data"}, nil)
	if result.ExitCode != 0 {
		return fmt.Errorf("rudolfs data directory not found on %s\n\nCreate storage directory:\n  ssh %s\n  mkdir -p /work/git/rudolfs/data",
			hostname, hostname)
	}

	if debug {
		fmt.Printf("  ✓ rudolfs configuration validated\n")
	}

	return nil
}

// getServerStartupInstructions returns server-specific startup instructions.
func getServerStartupInstructions(serverType, serverURL string) string {
	parsedURL, _ := url.Parse(serverURL)
	hostname := parsedURL.Hostname()

	switch serverType {
	case "lfs-test-server":
		return dedent.Dedent(fmt.Sprintf(`
			To start the LFS Test Server on %s:
			  ssh %s
			  export LFS_CONTENTPATH=/opt/lfs-test-server
			  ~/go/bin/lfs-test-server -addr :8080

			Or see: https://github.com/git-lfs/lfs-test-server
			`, hostname, hostname))

	case "giftless":
		return dedent.Dedent(fmt.Sprintf(`
			To start Giftless on %s:
			  ssh %s
			  cd /work/git/giftless
			  ./start-giftless.sh

			Or see: https://github.com/datopian/giftless
			`, hostname, hostname))

	case "rudolfs":
		return dedent.Dedent(fmt.Sprintf(`
			To start Rudolfs on %s:
			  ssh %s
			  cd /work/git/rudolfs
			  ./rudolfs --host 0.0.0.0 --port 8080

			Or see: https://github.com/jasonwhite/rudolfs
			`, hostname, hostname))

	default:
		return dedent.Dedent(fmt.Sprintf(`
			To start the LFS server on %s:
			  ssh %s
			  # Start your LFS server according to its documentation
			`, hostname, hostname))
	}
}

// checkServerConnectivity verifies that the LFS server is reachable with a short timeout.
// Returns a helpful error message if the server is not accessible.
func checkServerConnectivity(serverURL, serverType string, timeout time.Duration, debug bool) error {
	// Parse the server URL
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL '%s': %w", serverURL, err)
	}

	host := parsedURL.Host
	if host == "" {
		return fmt.Errorf("invalid server URL '%s': missing host", serverURL)
	}

	// Add default port if not specified
	if parsedURL.Port() == "" {
		switch parsedURL.Scheme {
		case "http":
			host = net.JoinHostPort(host, "80")
		case "https":
			host = net.JoinHostPort(host, "443")
		}
	}

	// First, try TCP connection with short timeout
	if debug {
		fmt.Printf("  Checking TCP connectivity to %s (timeout: %.0fs)...\n", host, timeout.Seconds())
	}
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		startupInstructions := getServerStartupInstructions(serverType, serverURL)
		return fmt.Errorf("LFS server not reachable at %s (timeout after %.0fs)\n\n%s\nPlease ensure:\n  1. The LFS server is running (see instructions above)\n  2. The hostname/IP is correct\n  3. Network connectivity is available\n\nError: %v",
			serverURL, timeout.Seconds(), startupInstructions, err)
	}
	conn.Close()

	// Try HTTP request to verify it's actually an HTTP server
	if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
		if debug {
			fmt.Printf("  Checking HTTP response from %s...\n", serverURL)
		}
		client := &http.Client{
			Timeout: timeout,
		}
		resp, err := client.Get(serverURL)
		if err != nil {
			startupInstructions := getServerStartupInstructions(serverType, serverURL)
			return fmt.Errorf("LFS server at %s is not responding to HTTP requests (timeout after %.0fs)\n\n%s\nPlease ensure the LFS server is running and configured correctly.\n\nError: %v",
				serverURL, timeout.Seconds(), startupInstructions, err)
		}
		resp.Body.Close()

		if debug {
			fmt.Printf("  HTTP response status: %d\n", resp.StatusCode)
		}
	}

	return nil
}

// generateReport generates a markdown report for the test run
func (r *Runner) generateReport() error {
	if r.Debug {
		fmt.Println("Generating test report...")
	}

	// Determine output path
	outputPath := r.ReportPath
	if outputPath == "" {
		// Default to WorkDir/report-{runID}.md
		outputPath = filepath.Join(r.WorkDir, fmt.Sprintf("report-%d.md", r.RunID))
	}

	// Generate report
	gen := report.NewGenerator(r.DB, r.RunID, outputPath)
	if err := gen.Generate(); err != nil {
		return err
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
