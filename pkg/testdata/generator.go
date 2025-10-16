package testdata

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileSpec describes a test file to copy
type FileSpec struct {
	Name       string
	SourcePath string
}

// CopyFile copies a single file to the destination
// Supports both local and remote sources (host:/path format)
func CopyFile(srcPath, destPath string, debug bool) error {
	// Check if source is remote
	if remotePath, isRemote := ParseRemotePath(srcPath); isRemote {
		return CopyRemoteFile(remotePath.Host, remotePath.Path, destPath, debug)
	}

	// Local file copy
	if debug {
		info, err := os.Stat(srcPath)
		if err == nil {
			fmt.Printf("  Copying %s (%s)\n", filepath.Base(destPath), FormatSize(info.Size()))
		}
	}

	// Create parent directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source file
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// CopyRemoteFile copies a file from a remote host using rsync over SSH
func CopyRemoteFile(host, remotePath, destPath string, debug bool) error {
	if debug {
		fmt.Printf("  Copying %s from %s via rsync\n", filepath.Base(destPath), host)
	}

	// Create parent directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Use rsync for efficient remote copying
	// -a: archive mode (preserves permissions, timestamps)
	// -q: quiet mode (unless debug)
	// -e ssh: use SSH
	args := []string{"-a", "-e", "ssh"}
	if !debug {
		args = append(args, "-q")
	}
	args = append(args, fmt.Sprintf("%s:%s", host, remotePath), destPath)

	cmd := exec.Command("rsync", args...)
	if debug {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync failed: %w", err)
	}

	return nil
}

// CopyFiles copies multiple test files
func CopyFiles(destDir string, specs []FileSpec, debug bool) error {
	if debug {
		fmt.Printf("Copying %d test files to %s\n", len(specs), destDir)
	}

	for _, spec := range specs {
		destPath := filepath.Join(destDir, spec.Name)
		if err := CopyFile(spec.SourcePath, destPath, debug); err != nil {
			return fmt.Errorf("failed to copy %s: %w", spec.Name, err)
		}
	}

	if debug {
		fmt.Printf("✓ Copied %d files\n", len(specs))
	}

	return nil
}

// RemotePath represents a remote path (host:/path)
type RemotePath struct {
	Host string
	Path string
}

// ParseRemotePath parses a path that may be remote (host:/path) or local
func ParseRemotePath(path string) (*RemotePath, bool) {
	// Check for remote format: host:/path
	if strings.Contains(path, ":") {
		parts := strings.SplitN(path, ":", 2)
		if len(parts) == 2 && !strings.HasPrefix(parts[0], "/") {
			// Check if this is a Windows drive letter (single letter before colon)
			if len(parts[0]) == 1 && parts[0][0] >= 'A' && parts[0][0] <= 'Z' ||
				len(parts[0]) == 1 && parts[0][0] >= 'a' && parts[0][0] <= 'z' {
				// This is a Windows path like C:/path
				return nil, false
			}
			// This looks like host:/path
			return &RemotePath{
				Host: parts[0],
				Path: parts[1],
			}, true
		}
	}
	return nil, false
}

// IsRemoteAccessible checks if a remote host is accessible via SSH
func IsRemoteAccessible(host string) error {
	cmd := exec.Command("ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes", host, "echo", "ok")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot connect to %s via SSH: %w", host, err)
	}
	return nil
}

// CheckRemoteDir checks if a directory exists on a remote host
func CheckRemoteDir(host, path string) error {
	cmd := exec.Command("ssh", host, "test", "-d", path)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("remote directory %s:%s does not exist", host, path)
	}
	return nil
}

// GetTestDataPath returns the path to the test data directory
// Searches in multiple locations, prioritizing LFS_TEST_DATA environment variable
// Supports remote paths in format: host:/path (accessed via SSH)
func GetTestDataPath() (string, error) {
	candidates := []string{
		os.Getenv("LFS_TEST_DATA"),
		"/mnt/f/work/git/git_lfs_test_data",
		"/work/git/git_lfs_test_data",
		"/home/mslinn/git_lfs_test_data",
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}

		// Check if this is a remote path
		if remotePath, isRemote := ParseRemotePath(path); isRemote {
			// Verify remote is accessible
			if err := IsRemoteAccessible(remotePath.Host); err != nil {
				continue // Try next candidate
			}
			// Verify remote directory exists
			if err := CheckRemoteDir(remotePath.Host, remotePath.Path); err != nil {
				continue // Try next candidate
			}
			return path, nil // Return the remote path as-is
		}

		// Local path - check if it exists
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("test data directory not found (searched: %v)", candidates)
}

// joinPath joins path components, handling both local and remote paths
func joinPath(base, component string) string {
	if remotePath, isRemote := ParseRemotePath(base); isRemote {
		// Remote path: join the path component and reconstruct host:/path
		joined := filepath.Join(remotePath.Path, component)
		return fmt.Sprintf("%s:%s", remotePath.Host, joined)
	}
	// Local path
	return filepath.Join(base, component)
}

// RealTestFiles returns the actual large test files from v1/
// These are the files described in the evaluation procedure:
// - 7 files totaling 1.3GB
// - File sizes: 103M - 308M
// - File types: pdf, m4v, mov, avi, ogg, zip
// Supports both local and remote test data paths
func RealTestFiles() ([]FileSpec, error) {
	basePath, err := GetTestDataPath()
	if err != nil {
		return nil, err
	}

	v1Path := joinPath(basePath, "v1")

	return []FileSpec{
		{Name: "pdf1.pdf", SourcePath: joinPath(v1Path, "pdf1.pdf")},
		{Name: "video1.m4v", SourcePath: joinPath(v1Path, "video1.m4v")},
		{Name: "video2.mov", SourcePath: joinPath(v1Path, "video2.mov")},
		{Name: "video3.avi", SourcePath: joinPath(v1Path, "video3.avi")},
		{Name: "video4.ogg", SourcePath: joinPath(v1Path, "video4.ogg")},
		{Name: "zip1.zip", SourcePath: joinPath(v1Path, "zip1.zip")},
		{Name: "zip2.zip", SourcePath: joinPath(v1Path, "zip2.zip")},
	}, nil
}

// RealTestFilesV2 returns the updated test files from v2/
// These are used for testing file modifications/updates:
// - 4 files totaling 1.1GB
// - Updated versions of some v1 files (larger sizes)
// Supports both local and remote test data paths
func RealTestFilesV2() ([]FileSpec, error) {
	basePath, err := GetTestDataPath()
	if err != nil {
		return nil, err
	}

	v2Path := joinPath(basePath, "v2")

	return []FileSpec{
		{Name: "pdf1.pdf", SourcePath: joinPath(v2Path, "pdf1.pdf")},       // 205M (was 103M)
		{Name: "video2.mov", SourcePath: joinPath(v2Path, "video2.mov")},   // 398M (was 238M)
		{Name: "video3.avi", SourcePath: joinPath(v2Path, "video3.avi")},   // 272M (was 150M)
		{Name: "zip1.zip", SourcePath: joinPath(v2Path, "zip1.zip")},       // 200M (was 308M)
	}, nil
}

// DeleteFile deletes a file from the destination directory
func DeleteFile(destDir, fileName string, debug bool) error {
	filePath := filepath.Join(destDir, fileName)

	if debug {
		fmt.Printf("  Deleting %s\n", fileName)
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// RenameFile renames a file in the destination directory
func RenameFile(destDir, oldName, newName string, debug bool) error {
	oldPath := filepath.Join(destDir, oldName)
	newPath := filepath.Join(destDir, newName)

	if debug {
		fmt.Printf("  Renaming %s to %s\n", oldName, newName)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return nil
}

// FormatSize formats a size in bytes as a human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// TotalSize calculates the total size by checking actual files
// Supports both local and remote file paths
func TotalSize(specs []FileSpec) (int64, error) {
	var total int64
	for _, spec := range specs {
		// Check if this is a remote path
		if remotePath, isRemote := ParseRemotePath(spec.SourcePath); isRemote {
			// Get size from remote file
			size, err := GetRemoteFileSize(remotePath.Host, remotePath.Path)
			if err != nil {
				return 0, fmt.Errorf("failed to get size of %s: %w", spec.SourcePath, err)
			}
			total += size
		} else {
			// Local file
			info, err := os.Stat(spec.SourcePath)
			if err != nil {
				return 0, fmt.Errorf("failed to stat %s: %w", spec.SourcePath, err)
			}
			total += info.Size()
		}
	}
	return total, nil
}

// GetRemoteFileSize gets the size of a file on a remote host via SSH
func GetRemoteFileSize(host, path string) (int64, error) {
	cmd := exec.Command("ssh", host, "stat", "-c", "%s", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to stat remote file: %w", err)
	}

	var size int64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &size); err != nil {
		return 0, fmt.Errorf("failed to parse file size: %w", err)
	}

	return size, nil
}
