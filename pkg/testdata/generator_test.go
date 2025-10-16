package testdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRemotePath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantRemote bool
		wantHost   string
		wantPath   string
	}{
		{
			name:       "remote path",
			path:       "gojira:/work/data",
			wantRemote: true,
			wantHost:   "gojira",
			wantPath:   "/work/data",
		},
		{
			name:       "remote path with user",
			path:       "user@host:/path/to/dir",
			wantRemote: true,
			wantHost:   "user@host",
			wantPath:   "/path/to/dir",
		},
		{
			name:       "local absolute path",
			path:       "/local/path",
			wantRemote: false,
		},
		{
			name:       "local relative path",
			path:       "relative/path",
			wantRemote: false,
		},
		{
			name:       "windows path",
			path:       "C:/Windows/Path",
			wantRemote: false,
		},
		{
			name:       "path with colon in filename",
			path:       "/path/file:with:colons",
			wantRemote: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remotePath, isRemote := ParseRemotePath(tt.path)
			if isRemote != tt.wantRemote {
				t.Errorf("ParseRemotePath(%q) isRemote = %v, want %v", tt.path, isRemote, tt.wantRemote)
			}
			if isRemote && remotePath != nil {
				if remotePath.Host != tt.wantHost {
					t.Errorf("ParseRemotePath(%q) host = %v, want %v", tt.path, remotePath.Host, tt.wantHost)
				}
				if remotePath.Path != tt.wantPath {
					t.Errorf("ParseRemotePath(%q) path = %v, want %v", tt.path, remotePath.Path, tt.wantPath)
				}
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	tests := []struct {
		name      string
		base      string
		component string
		want      string
	}{
		{
			name:      "local paths",
			base:      "/home/user",
			component: "documents",
			want:      "/home/user/documents",
		},
		{
			name:      "remote path",
			base:      "gojira:/work/data",
			component: "v1",
			want:      "gojira:/work/data/v1",
		},
		{
			name:      "remote path with user",
			base:      "user@host:/base",
			component: "subdir",
			want:      "user@host:/base/subdir",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinPath(tt.base, tt.component)
			if got != tt.want {
				t.Errorf("joinPath(%q, %q) = %v, want %v", tt.base, tt.component, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1048576, "1.00 MB"},
		{1572864, "1.50 MB"},
		{1073741824, "1.00 GB"},
		{1610612736, "1.50 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestCopyFile_Local(t *testing.T) {
	// Create temporary directories
	srcDir, err := os.MkdirTemp("", "src_test")
	if err != nil {
		t.Fatalf("Failed to create src temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "dst_test")
	if err != nil {
		t.Fatalf("Failed to create dst temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create a test file
	srcFile := filepath.Join(srcDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(dstDir, "copied.txt")
	if err := CopyFile(srcFile, dstFile, false); err != nil {
		t.Fatalf("CopyFile failed: %v", err)
	}

	// Verify the copy
	copiedContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != string(content) {
		t.Errorf("Copied content = %q, want %q", string(copiedContent), string(content))
	}
}

func TestCopyFiles(t *testing.T) {
	// Create temporary directories
	srcDir, err := os.MkdirTemp("", "src_test")
	if err != nil {
		t.Fatalf("Failed to create src temp dir: %v", err)
	}
	defer os.RemoveAll(srcDir)

	dstDir, err := os.MkdirTemp("", "dst_test")
	if err != nil {
		t.Fatalf("Failed to create dst temp dir: %v", err)
	}
	defer os.RemoveAll(dstDir)

	// Create test files
	specs := []FileSpec{
		{Name: "file1.txt", SourcePath: filepath.Join(srcDir, "file1.txt")},
		{Name: "file2.txt", SourcePath: filepath.Join(srcDir, "file2.txt")},
	}

	for _, spec := range specs {
		if err := os.WriteFile(spec.SourcePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Copy files
	if err := CopyFiles(dstDir, specs, false); err != nil {
		t.Fatalf("CopyFiles failed: %v", err)
	}

	// Verify all files were copied
	for _, spec := range specs {
		dstPath := filepath.Join(dstDir, spec.Name)
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			t.Errorf("File %s was not copied", spec.Name)
		}
	}
}

func TestDeleteFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "delete_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := "test.txt"
	filePath := filepath.Join(tempDir, testFile)
	if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Delete the file
	if err := DeleteFile(tempDir, testFile, false); err != nil {
		t.Fatalf("DeleteFile failed: %v", err)
	}

	// Verify deletion
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("File was not deleted")
	}
}

func TestRenameFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "rename_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	oldName := "old.txt"
	newName := "new.txt"
	oldPath := filepath.Join(tempDir, oldName)
	newPath := filepath.Join(tempDir, newName)

	if err := os.WriteFile(oldPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Rename the file
	if err := RenameFile(tempDir, oldName, newName, false); err != nil {
		t.Fatalf("RenameFile failed: %v", err)
	}

	// Verify old file doesn't exist
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("Old file still exists after rename")
	}

	// Verify new file exists
	if _, err := os.Stat(newPath); os.IsNotExist(err) {
		t.Error("New file doesn't exist after rename")
	}
}

func TestTotalSize_Local(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "size_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files with known sizes
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	content1 := []byte("12345")     // 5 bytes
	content2 := []byte("1234567890") // 10 bytes

	if err := os.WriteFile(file1, content1, 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, content2, 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	specs := []FileSpec{
		{Name: "file1.txt", SourcePath: file1},
		{Name: "file2.txt", SourcePath: file2},
	}

	// Calculate total size
	total, err := TotalSize(specs)
	if err != nil {
		t.Fatalf("TotalSize failed: %v", err)
	}

	expectedTotal := int64(15) // 5 + 10
	if total != expectedTotal {
		t.Errorf("TotalSize() = %d, want %d", total, expectedTotal)
	}
}

func TestGetTestDataPath_EnvPriority(t *testing.T) {
	// Save original environment
	orig := os.Getenv("LFS_TEST_DATA")
	defer os.Setenv("LFS_TEST_DATA", orig)

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "testdata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set environment variable to temp dir
	os.Setenv("LFS_TEST_DATA", tempDir)

	// Get test data path
	path, err := GetTestDataPath()
	if err != nil {
		t.Fatalf("GetTestDataPath() failed: %v", err)
	}

	// Verify it returns the env var path
	if path != tempDir {
		t.Errorf("GetTestDataPath() = %v, want %v (from LFS_TEST_DATA)", path, tempDir)
	}
}

func TestRealTestFiles_Structure(t *testing.T) {
	// Save original environment
	orig := os.Getenv("LFS_TEST_DATA")
	defer os.Setenv("LFS_TEST_DATA", orig)

	// Create a temporary test data structure
	tempDir, err := os.MkdirTemp("", "testdata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create v1 directory
	v1Dir := filepath.Join(tempDir, "v1")
	if err := os.MkdirAll(v1Dir, 0755); err != nil {
		t.Fatalf("Failed to create v1 dir: %v", err)
	}

	// Create dummy test files with identifiable content
	testFiles := []string{"pdf1.pdf", "video1.m4v", "video2.mov", "video3.avi", "video4.ogg", "zip1.zip", "zip2.zip"}
	for _, name := range testFiles {
		path := filepath.Join(v1Dir, name)
		// Write identifiable content so we can verify source
		content := []byte("v1_content_" + name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Set environment to use temp dir
	os.Setenv("LFS_TEST_DATA", tempDir)

	// Get test files
	specs, err := RealTestFiles()
	if err != nil {
		t.Fatalf("RealTestFiles() failed: %v", err)
	}

	// Verify we got the expected number of files
	expectedCount := 7
	if len(specs) != expectedCount {
		t.Errorf("RealTestFiles() returned %d files, want %d", len(specs), expectedCount)
	}

	// Verify each file has proper structure
	for _, spec := range specs {
		if spec.Name == "" {
			t.Error("FileSpec has empty Name")
		}
		if spec.SourcePath == "" {
			t.Error("FileSpec has empty SourcePath")
		}
	}
}

func TestRealTestFiles_SourceFromV1(t *testing.T) {
	// Save original environment
	orig := os.Getenv("LFS_TEST_DATA")
	defer os.Setenv("LFS_TEST_DATA", orig)

	// Create a temporary test data structure
	tempDir, err := os.MkdirTemp("", "testdata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create v1 directory with distinct content
	v1Dir := filepath.Join(tempDir, "v1")
	if err := os.MkdirAll(v1Dir, 0755); err != nil {
		t.Fatalf("Failed to create v1 dir: %v", err)
	}

	expectedFiles := map[string]string{
		"pdf1.pdf":    "v1_pdf_content",
		"video1.m4v":  "v1_video1_content",
		"video2.mov":  "v1_video2_content",
		"video3.avi":  "v1_video3_content",
		"video4.ogg":  "v1_video4_content",
		"zip1.zip":    "v1_zip1_content",
		"zip2.zip":    "v1_zip2_content",
	}

	// Create test files with specific content
	for name, content := range expectedFiles {
		path := filepath.Join(v1Dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Set environment to use temp dir
	os.Setenv("LFS_TEST_DATA", tempDir)

	// Get test files
	specs, err := RealTestFiles()
	if err != nil {
		t.Fatalf("RealTestFiles() failed: %v", err)
	}

	// Verify each file's source path points to v1
	for _, spec := range specs {
		// Check that source path contains /v1/
		if !contains(spec.SourcePath, "/v1/") && !contains(spec.SourcePath, "\\v1\\") {
			t.Errorf("File %s source path %s doesn't contain /v1/", spec.Name, spec.SourcePath)
		}

		// Verify the file actually exists and has correct content
		content, err := os.ReadFile(spec.SourcePath)
		if err != nil {
			t.Errorf("Failed to read source file %s: %v", spec.SourcePath, err)
			continue
		}

		expectedContent := expectedFiles[spec.Name]
		if string(content) != expectedContent {
			t.Errorf("File %s has content %q, want %q (verifying it came from v1)",
				spec.Name, string(content), expectedContent)
		}
	}
}

func TestRealTestFilesV2_Structure(t *testing.T) {
	// Save original environment
	orig := os.Getenv("LFS_TEST_DATA")
	defer os.Setenv("LFS_TEST_DATA", orig)

	// Create a temporary test data structure
	tempDir, err := os.MkdirTemp("", "testdata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create v2 directory
	v2Dir := filepath.Join(tempDir, "v2")
	if err := os.MkdirAll(v2Dir, 0755); err != nil {
		t.Fatalf("Failed to create v2 dir: %v", err)
	}

	// Create dummy test files
	testFiles := []string{"pdf1.pdf", "video2.mov", "video3.avi", "zip1.zip"}
	for _, name := range testFiles {
		path := filepath.Join(v2Dir, name)
		if err := os.WriteFile(path, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}

	// Set environment to use temp dir
	os.Setenv("LFS_TEST_DATA", tempDir)

	// Get test files
	specs, err := RealTestFilesV2()
	if err != nil {
		t.Fatalf("RealTestFilesV2() failed: %v", err)
	}

	// Verify we got the expected number of files
	expectedCount := 4
	if len(specs) != expectedCount {
		t.Errorf("RealTestFilesV2() returned %d files, want %d", len(specs), expectedCount)
	}

	// Verify each file has proper structure
	for _, spec := range specs {
		if spec.Name == "" {
			t.Error("FileSpec has empty Name")
		}
		if spec.SourcePath == "" {
			t.Error("FileSpec has empty SourcePath")
		}
	}
}

func TestRealTestFilesV2_SourceFromV2(t *testing.T) {
	// Save original environment
	orig := os.Getenv("LFS_TEST_DATA")
	defer os.Setenv("LFS_TEST_DATA", orig)

	// Create a temporary test data structure
	tempDir, err := os.MkdirTemp("", "testdata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create both v1 and v2 directories to ensure v2 files come from v2
	v1Dir := filepath.Join(tempDir, "v1")
	v2Dir := filepath.Join(tempDir, "v2")
	if err := os.MkdirAll(v1Dir, 0755); err != nil {
		t.Fatalf("Failed to create v1 dir: %v", err)
	}
	if err := os.MkdirAll(v2Dir, 0755); err != nil {
		t.Fatalf("Failed to create v2 dir: %v", err)
	}

	// Create v1 versions with v1-specific content
	v1Files := map[string]string{
		"pdf1.pdf":    "v1_pdf_content",
		"video2.mov":  "v1_video2_content",
		"video3.avi":  "v1_video3_content",
		"zip1.zip":    "v1_zip1_content",
	}
	for name, content := range v1Files {
		path := filepath.Join(v1Dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create v1 test file %s: %v", name, err)
		}
	}

	// Create v2 versions with v2-specific content (updated versions)
	v2Files := map[string]string{
		"pdf1.pdf":    "v2_pdf_content_updated",
		"video2.mov":  "v2_video2_content_updated",
		"video3.avi":  "v2_video3_content_updated",
		"zip1.zip":    "v2_zip1_content_updated",
	}
	for name, content := range v2Files {
		path := filepath.Join(v2Dir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create v2 test file %s: %v", name, err)
		}
	}

	// Set environment to use temp dir
	os.Setenv("LFS_TEST_DATA", tempDir)

	// Get v2 test files
	specs, err := RealTestFilesV2()
	if err != nil {
		t.Fatalf("RealTestFilesV2() failed: %v", err)
	}

	// Verify each file's source path points to v2
	for _, spec := range specs {
		// Check that source path contains /v2/
		if !contains(spec.SourcePath, "/v2/") && !contains(spec.SourcePath, "\\v2\\") {
			t.Errorf("File %s source path %s doesn't contain /v2/", spec.Name, spec.SourcePath)
		}

		// Verify the file actually exists and has v2 content (not v1)
		content, err := os.ReadFile(spec.SourcePath)
		if err != nil {
			t.Errorf("Failed to read source file %s: %v", spec.SourcePath, err)
			continue
		}

		expectedContent := v2Files[spec.Name]
		if string(content) != expectedContent {
			t.Errorf("File %s has content %q, want %q (verifying it came from v2, not v1)",
				spec.Name, string(content), expectedContent)
		}

		// Double-check it's NOT the v1 content
		v1Content := v1Files[spec.Name]
		if string(content) == v1Content {
			t.Errorf("File %s has v1 content %q, but should have v2 content (source verification failed)",
				spec.Name, v1Content)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
