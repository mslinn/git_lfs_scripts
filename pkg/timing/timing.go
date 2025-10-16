package timing

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// Result contains the results of a timed command execution
type Result struct {
	Command    string
	Args       []string
	DurationMs int64
	Stdout     string
	Stderr     string
	ExitCode   int
	Error      error
}

// Options configures command execution
type Options struct {
	Dir     string        // Working directory
	Timeout time.Duration // Command timeout (0 for no timeout)
	Debug   bool          // Enable debug output
}

// Run executes a command and measures its execution time with millisecond precision
func Run(command string, args []string, opts *Options) *Result {
	if opts == nil {
		opts = &Options{}
	}

	result := &Result{
		Command: command,
		Args:    args,
	}

	// Create context with timeout if specified
	var ctx context.Context
	var cancel context.CancelFunc
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	} else {
		ctx = context.Background()
	}

	// Create command
	cmd := exec.CommandContext(ctx, command, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Time the execution
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result.DurationMs = duration.Milliseconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// Success returns true if the command executed successfully
func (r *Result) Success() bool {
	return r.ExitCode == 0 && r.Error == nil
}

// String returns a human-readable summary of the result
func (r *Result) String() string {
	status := "success"
	if !r.Success() {
		status = fmt.Sprintf("failed (exit code %d)", r.ExitCode)
	}

	return fmt.Sprintf("%s %v: %s (%.3fs)",
		r.Command,
		r.Args,
		status,
		float64(r.DurationMs)/1000.0,
	)
}

// DebugString returns a detailed debug output
func (r *Result) DebugString() string {
	output := r.String() + "\n"

	if r.Stdout != "" {
		output += fmt.Sprintf("STDOUT:\n%s\n", r.Stdout)
	}

	if r.Stderr != "" {
		output += fmt.Sprintf("STDERR:\n%s\n", r.Stderr)
	}

	if r.Error != nil {
		output += fmt.Sprintf("ERROR: %v\n", r.Error)
	}

	return output
}
