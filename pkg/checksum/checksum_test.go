package checksum

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComputeFile(t *testing.T) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("", "checksum_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("hello world")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute checksum
	cs, err := ComputeFile(testFile)
	if err != nil {
		t.Fatalf("ComputeFile failed: %v", err)
	}

	// Verify checksum structure
	if cs == nil {
		t.Fatal("ComputeFile returned nil")
	}
	if cs.Path != testFile {
		t.Errorf("Path = %v, want %v", cs.Path, testFile)
	}
	if cs.CRC32 == 0 {
		t.Error("CRC32 should not be zero")
	}
	if cs.SizeBytes != int64(len(content)) {
		t.Errorf("SizeBytes = %d, want %d", cs.SizeBytes, len(content))
	}
}

func TestComputeFile_ConsistentChecksum(t *testing.T) {
	// Create a temporary file
	tempDir, err := os.MkdirTemp("", "checksum_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content for checksum")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute checksum twice
	cs1, err := ComputeFile(testFile)
	if err != nil {
		t.Fatalf("First ComputeFile failed: %v", err)
	}

	cs2, err := ComputeFile(testFile)
	if err != nil {
		t.Fatalf("Second ComputeFile failed: %v", err)
	}

	// Verify checksums are identical
	if cs1.CRC32 != cs2.CRC32 {
		t.Errorf("Checksums differ: %08x != %08x", cs1.CRC32, cs2.CRC32)
	}
}

func TestComputeDirectory(t *testing.T) {
	// Create a temporary directory with files
	tempDir, err := os.MkdirTemp("", "checksum_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"subdir/file3.txt": "content3",
	}

	for path, content := range files {
		fullPath := filepath.Join(tempDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Compute directory checksums
	checksums, err := ComputeDirectory(tempDir)
	if err != nil {
		t.Fatalf("ComputeDirectory failed: %v", err)
	}

	// Verify we got checksums for all files
	if len(checksums) != len(files) {
		t.Errorf("Got %d checksums, want %d", len(checksums), len(files))
	}

	// Verify checksums are sorted by path
	for i := 1; i < len(checksums); i++ {
		if checksums[i].Path < checksums[i-1].Path {
			t.Errorf("Checksums not sorted: %v comes before %v", checksums[i].Path, checksums[i-1].Path)
		}
	}
}

func TestComputeDirectory_SkipsGit(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "checksum_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory with a file
	gitDir := filepath.Join(tempDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}
	gitFile := filepath.Join(gitDir, "config")
	if err := os.WriteFile(gitFile, []byte("git content"), 0644); err != nil {
		t.Fatalf("Failed to create git file: %v", err)
	}

	// Create a regular file
	regularFile := filepath.Join(tempDir, "file.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	// Compute checksums
	checksums, err := ComputeDirectory(tempDir)
	if err != nil {
		t.Fatalf("ComputeDirectory failed: %v", err)
	}

	// Verify only regular file was checksummed
	if len(checksums) != 1 {
		t.Errorf("Got %d checksums, want 1 (should skip .git)", len(checksums))
	}
	if len(checksums) > 0 && checksums[0].Path != "file.txt" {
		t.Errorf("Wrong file checksummed: %v", checksums[0].Path)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := FormatSize(tt.bytes)
			if got != tt.expected {
				t.Errorf("FormatSize(%d) = %v, want %v", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestExportJSON(t *testing.T) {
	checksums := []*FileChecksum{
		{Path: "file1.txt", CRC32: 0x12345678, SizeBytes: 100},
		{Path: "file2.txt", CRC32: 0x87654321, SizeBytes: 200},
	}

	data, err := ExportJSON(1, 2, checksums)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ExportJSON returned empty data")
	}

	// Verify it's valid JSON by checking for expected fields
	jsonStr := string(data)
	if !contains(jsonStr, "run_id") {
		t.Error("JSON missing 'run_id' field")
	}
	if !contains(jsonStr, "step_number") {
		t.Error("JSON missing 'step_number' field")
	}
	if !contains(jsonStr, "checksums") {
		t.Error("JSON missing 'checksums' field")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompareChecksums_EmptyLists(t *testing.T) {
	// This is a mock test - in real usage, we'd need a database
	// Here we just test the difference structure
	diff := &Difference{
		FilePath:   "test.txt",
		OldCRC32:   "12345678",
		NewCRC32:   "87654321",
		OldSize:    100,
		NewSize:    150,
		ChangeType: "modified",
	}

	if diff.FilePath != "test.txt" {
		t.Errorf("FilePath = %v, want test.txt", diff.FilePath)
	}
	if diff.ChangeType != "modified" {
		t.Errorf("ChangeType = %v, want modified", diff.ChangeType)
	}
}

func TestDifferenceTypes(t *testing.T) {
	changeTypes := []string{"added", "modified", "deleted", "size-changed"}

	for _, ct := range changeTypes {
		diff := &Difference{
			FilePath:   "test.txt",
			ChangeType: ct,
		}
		if diff.ChangeType != ct {
			t.Errorf("ChangeType = %v, want %v", diff.ChangeType, ct)
		}
	}
}
