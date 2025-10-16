package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mslinn/git_lfs_scripts/pkg/database"
	"github.com/mslinn/git_lfs_scripts/pkg/timing"
)

// Context holds the execution context for git operations
type Context struct {
	DB         *database.DB
	RunID      int64
	StepNumber int
	Debug      bool
	WorkDir    string // Working directory for operations
}

// recordOperation records a git operation in the database
func (ctx *Context) recordOperation(opType, command string, result *timing.Result) error {
	if ctx.DB == nil {
		return nil // Skip if no database
	}

	status := "success"
	errorMsg := ""
	if result.Error != nil {
		status = "failed"
		errorMsg = result.Error.Error()
	} else if result.ExitCode != 0 {
		status = "failed"
		errorMsg = fmt.Sprintf("exit code %d: %s", result.ExitCode, result.Stderr)
	}

	op := &database.Operation{
		RunID:       ctx.RunID,
		StepNumber:  ctx.StepNumber,
		Operation:   opType,
		StartedAt:   time.Now().Add(-time.Duration(result.DurationMs) * time.Millisecond),
		DurationMs:  result.DurationMs,
		FileCount:   nil, // TODO: extract from output
		TotalBytes:  nil, // TODO: extract from output
		Status:      status,
		Error:       errorMsg,
	}

	return ctx.DB.CreateOperation(op)
}

// Clone clones a git repository
func (ctx *Context) Clone(url, destDir string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Cloning %s to %s\n", ctx.StepNumber, url, destDir)
	}

	// Remove destination if it exists
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove existing directory: %w", err)
	}

	// Create parent directory
	parent := filepath.Dir(destDir)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// Run git clone
	result := timing.Run("git", []string{"clone", url, destDir}, nil)
	if err := ctx.recordOperation("clone", fmt.Sprintf("git clone %s", url), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git clone failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git clone failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Cloned in %dms\n", result.DurationMs)
	}

	return nil
}

// InitRepo initializes a new git repository
func (ctx *Context) InitRepo(dir string, bare bool) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Initializing git repository in %s (bare=%v)\n", ctx.StepNumber, dir, bare)
	}

	// Create directory
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Run git init
	args := []string{"init"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, dir)

	result := timing.Run("git", args, nil)
	if err := ctx.recordOperation("init", fmt.Sprintf("git init %s", dir), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git init failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git init failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Initialized in %dms\n", result.DurationMs)
	}

	return nil
}

// Add stages files for commit
func (ctx *Context) Add(repoDir string, paths ...string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Adding files: %v\n", ctx.StepNumber, paths)
	}

	args := append([]string{"-C", repoDir, "add"}, paths...)
	result := timing.Run("git", args, nil)

	if err := ctx.recordOperation("add", fmt.Sprintf("git add %s", strings.Join(paths, " ")), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git add failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git add failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Added in %dms\n", result.DurationMs)
	}

	return nil
}

// Commit creates a commit
func (ctx *Context) Commit(repoDir, message string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Committing: %s\n", ctx.StepNumber, message)
	}

	result := timing.Run("git", []string{"-C", repoDir, "commit", "-m", message}, nil)

	if err := ctx.recordOperation("commit", "git commit", result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git commit failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git commit failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Committed in %dms\n", result.DurationMs)
	}

	return nil
}

// Push pushes commits to remote
func (ctx *Context) Push(repoDir, remote, branch string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Pushing to %s/%s\n", ctx.StepNumber, remote, branch)
	}

	result := timing.Run("git", []string{"-C", repoDir, "push", remote, branch}, nil)

	if err := ctx.recordOperation("push", fmt.Sprintf("git push %s %s", remote, branch), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git push failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git push failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Pushed in %dms\n", result.DurationMs)
	}

	return nil
}

// Pull pulls commits from remote
func (ctx *Context) Pull(repoDir string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Pulling changes\n", ctx.StepNumber)
	}

	result := timing.Run("git", []string{"-C", repoDir, "pull"}, nil)

	if err := ctx.recordOperation("pull", "git pull", result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git pull failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git pull failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Pulled in %dms\n", result.DurationMs)
	}

	return nil
}

// ConfigUser sets git user configuration for a repository
func (ctx *Context) ConfigUser(repoDir, name, email string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Configuring git user: %s <%s>\n", ctx.StepNumber, name, email)
	}

	// Set user.name
	result1 := timing.Run("git", []string{"-C", repoDir, "config", "user.name", name}, nil)
	if result1.Error != nil || result1.ExitCode != 0 {
		return fmt.Errorf("failed to set user.name: %v", result1.Error)
	}

	// Set user.email
	result2 := timing.Run("git", []string{"-C", repoDir, "config", "user.email", email}, nil)
	if result2.Error != nil || result2.ExitCode != 0 {
		return fmt.Errorf("failed to set user.email: %v", result2.Error)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Configured user\n")
	}

	return nil
}

// ConfigureLFSURL sets the LFS server URL in .lfsconfig
func (ctx *Context) ConfigureLFSURL(repoDir, url string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Configuring LFS URL: %s\n", ctx.StepNumber, url)
	}

	lfsConfigPath := filepath.Join(repoDir, ".lfsconfig")
	content := fmt.Sprintf("[lfs]\n\turl = %s\n", url)

	if err := os.WriteFile(lfsConfigPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write .lfsconfig: %w", err)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Created .lfsconfig\n")
	}

	return nil
}

// CreateGitHubRepo creates a private GitHub repository using gh CLI
// Returns the clone URL for the created repository
func (ctx *Context) CreateGitHubRepo(repoName string, force bool) (string, error) {
	if ctx.Debug {
		fmt.Printf("[Step %d] Creating GitHub repository: %s\n", ctx.StepNumber, repoName)
	}

	// Check if gh CLI is available
	checkResult := timing.Run("gh", []string{"--version"}, nil)
	if checkResult.Error != nil || checkResult.ExitCode != 0 {
		return "", fmt.Errorf("gh CLI not available - install with: sudo apt install gh")
	}

	// Delete existing repo if force flag is set
	if force {
		if ctx.Debug {
			fmt.Printf("  Checking if repo already exists...\n")
		}
		deleteResult := timing.Run("gh", []string{"repo", "delete", repoName, "--yes"}, nil)
		if deleteResult.ExitCode == 0 && ctx.Debug {
			fmt.Printf("  ✓ Deleted existing repository\n")
		}
	}

	// Create private repository
	args := []string{"repo", "create", repoName, "--private"}
	result := timing.Run("gh", args, nil)

	if err := ctx.recordOperation("gh-create-repo", fmt.Sprintf("gh repo create %s", repoName), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return "", fmt.Errorf("gh repo create failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("gh repo create failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	// Get the clone URL
	cloneURL := fmt.Sprintf("https://github.com/%s.git", repoName)

	if ctx.Debug {
		fmt.Printf("  ✓ Created GitHub repository in %dms\n", result.DurationMs)
		fmt.Printf("  Clone URL: %s\n", cloneURL)
	}

	return cloneURL, nil
}

// AddRemote adds a git remote to a repository
func (ctx *Context) AddRemote(repoDir, remoteName, url string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Adding remote '%s': %s\n", ctx.StepNumber, remoteName, url)
	}

	result := timing.Run("git", []string{"-C", repoDir, "remote", "add", remoteName, url}, nil)

	if err := ctx.recordOperation("add-remote", fmt.Sprintf("git remote add %s", remoteName), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git remote add failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git remote add failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Added remote in %dms\n", result.DurationMs)
	}

	return nil
}

// LFSInstall installs git-lfs hooks in a repository
func (ctx *Context) LFSInstall(repoDir string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Installing git-lfs hooks\n", ctx.StepNumber)
	}

	result := timing.Run("git", []string{"-C", repoDir, "lfs", "install"}, nil)

	if err := ctx.recordOperation("lfs-install", "git lfs install", result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git lfs install failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git lfs install failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Installed git-lfs in %dms\n", result.DurationMs)
	}

	return nil
}

// LFSTrack adds a pattern to git-lfs tracking
func (ctx *Context) LFSTrack(repoDir, pattern string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Tracking pattern with git-lfs: %s\n", ctx.StepNumber, pattern)
	}

	result := timing.Run("git", []string{"-C", repoDir, "lfs", "track", pattern}, nil)

	if err := ctx.recordOperation("lfs-track", fmt.Sprintf("git lfs track %s", pattern), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git lfs track failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git lfs track failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Tracked %s in %dms\n", pattern, result.DurationMs)
	}

	return nil
}

// LFSUntrack removes a pattern from git-lfs tracking
func (ctx *Context) LFSUntrack(repoDir, pattern string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Untracking pattern from git-lfs: %s\n", ctx.StepNumber, pattern)
	}

	result := timing.Run("git", []string{"-C", repoDir, "lfs", "untrack", pattern}, nil)

	if err := ctx.recordOperation("lfs-untrack", fmt.Sprintf("git lfs untrack %s", pattern), result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git lfs untrack failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git lfs untrack failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Untracked %s in %dms\n", pattern, result.DurationMs)
	}

	return nil
}

// LFSMigrate migrates files out of LFS back to regular git
func (ctx *Context) LFSMigrate(repoDir string) error {
	if ctx.Debug {
		fmt.Printf("[Step %d] Migrating files out of LFS\n", ctx.StepNumber)
	}

	// Use git lfs migrate export to move files out of LFS
	result := timing.Run("git", []string{"-C", repoDir, "lfs", "migrate", "export", "--include=*", "--everything"}, nil)

	if err := ctx.recordOperation("lfs-migrate", "git lfs migrate export", result); err != nil {
		if ctx.Debug {
			fmt.Printf("  Warning: failed to record operation: %v\n", err)
		}
	}

	if result.Error != nil {
		return fmt.Errorf("git lfs migrate failed: %w", result.Error)
	}

	if result.ExitCode != 0 {
		return fmt.Errorf("git lfs migrate failed (exit %d): %s", result.ExitCode, result.Stderr)
	}

	if ctx.Debug {
		fmt.Printf("  ✓ Migrated files in %dms\n", result.DurationMs)
	}

	return nil
}
