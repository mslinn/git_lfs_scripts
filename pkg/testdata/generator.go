package testdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileSpec describes a test file to copy
type FileSpec struct {
	Name       string
	SourcePath string
}

// CopyFile copies a single file to the destination
func CopyFile(srcPath, destPath string, debug bool) error {
	if debug {
		info, _ := os.Stat(srcPath)
		fmt.Printf("  Copying %s (%s)\n", filepath.Base(destPath), FormatSize(info.Size()))
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

// GetTestDataPath returns the path to the test data directory
// Searches in multiple locations
func GetTestDataPath() (string, error) {
	candidates := []string{
		"/mnt/f/work/git/git_lfs_test_data",
		"/work/git/git_lfs_test_data",
		"/home/mslinn/git_lfs_test_data",
		os.Getenv("LFS_TEST_DATA"),
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("test data directory not found (searched: %v)", candidates)
}

// RealTestFiles returns the actual large test files from v1/
// These are the files described in the evaluation procedure:
// - 7 files totaling 1.3GB
// - File sizes: 103M - 308M
// - File types: pdf, m4v, mov, avi, ogg, zip
func RealTestFiles() ([]FileSpec, error) {
	basePath, err := GetTestDataPath()
	if err != nil {
		return nil, err
	}

	v1Path := filepath.Join(basePath, "v1")

	return []FileSpec{
		{Name: "pdf1.pdf", SourcePath: filepath.Join(v1Path, "pdf1.pdf")},
		{Name: "video1.m4v", SourcePath: filepath.Join(v1Path, "video1.m4v")},
		{Name: "video2.mov", SourcePath: filepath.Join(v1Path, "video2.mov")},
		{Name: "video3.avi", SourcePath: filepath.Join(v1Path, "video3.avi")},
		{Name: "video4.ogg", SourcePath: filepath.Join(v1Path, "video4.ogg")},
		{Name: "zip1.zip", SourcePath: filepath.Join(v1Path, "zip1.zip")},
		{Name: "zip2.zip", SourcePath: filepath.Join(v1Path, "zip2.zip")},
	}, nil
}

// RealTestFilesV2 returns the updated test files from v2/
// These are used for testing file modifications/updates:
// - 4 files totaling 1.1GB
// - Updated versions of some v1 files (larger sizes)
func RealTestFilesV2() ([]FileSpec, error) {
	basePath, err := GetTestDataPath()
	if err != nil {
		return nil, err
	}

	v2Path := filepath.Join(basePath, "v2")

	return []FileSpec{
		{Name: "pdf1.pdf", SourcePath: filepath.Join(v2Path, "pdf1.pdf")},       // 205M (was 103M)
		{Name: "video2.mov", SourcePath: filepath.Join(v2Path, "video2.mov")},   // 398M (was 238M)
		{Name: "video3.avi", SourcePath: filepath.Join(v2Path, "video3.avi")},   // 272M (was 150M)
		{Name: "zip1.zip", SourcePath: filepath.Join(v2Path, "zip1.zip")},       // 200M (was 308M)
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
func TotalSize(specs []FileSpec) (int64, error) {
	var total int64
	for _, spec := range specs {
		info, err := os.Stat(spec.SourcePath)
		if err != nil {
			return 0, fmt.Errorf("failed to stat %s: %w", spec.SourcePath, err)
		}
		total += info.Size()
	}
	return total, nil
}
