package timing

import (
	"os"
	"testing"
	"time"
)

func TestRun_Success(t *testing.T) {
	// Run a simple command
	result := Run("echo", []string{"hello"}, nil)

	if result == nil {
		t.Fatal("Run returned nil")
	}

	if result.Error != nil {
		t.Errorf("Run failed: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	if result.DurationMs <= 0 {
		t.Error("DurationMs should be positive")
	}

	if len(result.Stdout) == 0 {
		t.Error("Stdout should not be empty")
	}
}

func TestRun_WithOptions(t *testing.T) {
	// Run with custom options
	opts := &Options{
		Debug: true,
	}

	// Use a command that prints text
	result := Run("echo", []string{"test_output"}, opts)

	if result.Error != nil {
		t.Fatalf("Run failed: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	// Output should contain our text
	if len(result.Stdout) == 0 {
		t.Error("Stdout should contain output")
	}
}

func TestRun_NonZeroExit(t *testing.T) {
	// Run a command that will fail
	result := Run("sh", []string{"-c", "exit 42"}, nil)

	if result == nil {
		t.Fatal("Run returned nil")
	}

	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}

	if result.DurationMs < 0 {
		t.Errorf("DurationMs = %d, should not be negative even for failed commands", result.DurationMs)
	}
}

func TestRun_NonexistentCommand(t *testing.T) {
	// Try to run a command that doesn't exist
	result := Run("nonexistent_command_xyz", []string{}, nil)

	if result == nil {
		t.Fatal("Run returned nil")
	}

	if result.Error == nil {
		t.Error("Expected error for nonexistent command")
	}

	if result.ExitCode == 0 {
		t.Error("ExitCode should not be 0 for failed command")
	}
}

func TestRun_StderrCapture(t *testing.T) {
	// Run a command that writes to stderr
	result := Run("sh", []string{"-c", "echo error_message >&2"}, nil)

	if result.Error != nil && result.ExitCode != 0 {
		t.Fatalf("Run failed unexpectedly: %v", result.Error)
	}

	if len(result.Stderr) == 0 {
		t.Error("Stderr should not be empty")
	}
}

func TestRun_Timing(t *testing.T) {
	// Run a command that takes a known amount of time
	start := time.Now()
	result := Run("sleep", []string{"0.1"}, nil) // Sleep for 100ms
	elapsed := time.Since(start)

	if result.Error != nil {
		t.Fatalf("Run failed: %v", result.Error)
	}

	// Verify timing is reasonable (should be at least 100ms)
	if result.DurationMs < 100 {
		t.Errorf("DurationMs = %d, want >= 100", result.DurationMs)
	}

	// Verify elapsed time matches roughly
	if elapsed.Milliseconds() < 100 {
		t.Errorf("Elapsed time = %dms, want >= 100ms", elapsed.Milliseconds())
	}

	// Duration should be reasonably close to actual elapsed time (within 50ms)
	diff := int64(result.DurationMs) - elapsed.Milliseconds()
	if diff < -50 || diff > 50 {
		t.Logf("Warning: DurationMs (%d) and elapsed (%d) differ by %dms",
			result.DurationMs, elapsed.Milliseconds(), diff)
	}
}

func TestResult_Structure(t *testing.T) {
	result := &Result{
		Stdout:     "output",
		Stderr:     "errors",
		ExitCode:   1,
		DurationMs: 100,
		Error:      nil,
	}

	if result.Stdout != "output" {
		t.Errorf("Stdout = %v, want output", result.Stdout)
	}
	if result.Stderr != "errors" {
		t.Errorf("Stderr = %v, want errors", result.Stderr)
	}
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.DurationMs != 100 {
		t.Errorf("DurationMs = %d, want 100", result.DurationMs)
	}
	if result.Error != nil {
		t.Errorf("Error = %v, want nil", result.Error)
	}
}

func TestRun_LargeOutput(t *testing.T) {
	// Generate a large amount of output
	result := Run("sh", []string{"-c", "for i in $(seq 1 1000); do echo line$i; done"}, nil)

	if result.Error != nil {
		t.Fatalf("Run failed: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}

	// Verify we captured a lot of output
	if len(result.Stdout) < 5000 {
		t.Errorf("Stdout length = %d, expected > 5000 (for 1000 lines)", len(result.Stdout))
	}
}

func TestRun_WithWorkingDirectory(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "timing_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testFile := "test.txt"
	testPath := tempDir + "/" + testFile
	if err := os.WriteFile(testPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to temp directory, run ls, and verify output
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	result := Run("ls", []string{}, nil)
	if result.Error != nil {
		t.Fatalf("Run failed: %v", result.Error)
	}

	// Output should contain our test file
	if !contains(result.Stdout, testFile) {
		t.Errorf("Output doesn't contain %s: %s", testFile, result.Stdout)
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

func TestRun_EmptyCommand(t *testing.T) {
	// Test with empty command (should fail gracefully)
	result := Run("", []string{}, nil)

	if result == nil {
		t.Fatal("Run returned nil for empty command")
	}

	if result.Error == nil {
		t.Error("Expected error for empty command")
	}
}

func TestRun_NilArgs(t *testing.T) {
	// Test with nil args (should work)
	result := Run("echo", nil, nil)

	if result.Error != nil {
		t.Errorf("Run failed with nil args: %v", result.Error)
	}

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}
