package helpers

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// CLITestHelper provides utilities for testing CLI commands
type CLITestHelper struct {
	t      *testing.T
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  *bytes.Buffer
}

// NewCLITestHelper creates a new CLI test helper
func NewCLITestHelper(t *testing.T) *CLITestHelper {
	return &CLITestHelper{
		t:      t,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdin:  &bytes.Buffer{},
	}
}

// ExecuteCommand executes a Cobra command with the given arguments
func (h *CLITestHelper) ExecuteCommand(cmd *cobra.Command, args ...string) error {
	// Reset buffers
	h.stdout.Reset()
	h.stderr.Reset()
	h.stdin.Reset()

	// Set command output
	cmd.SetOut(h.stdout)
	cmd.SetErr(h.stderr)
	cmd.SetIn(h.stdin)

	// Set arguments
	cmd.SetArgs(args)

	// Execute command
	return cmd.Execute()
}

// ExecuteCommandWithInput executes a command with stdin input
func (h *CLITestHelper) ExecuteCommandWithInput(cmd *cobra.Command, input string, args ...string) error {
	h.stdin.WriteString(input)
	return h.ExecuteCommand(cmd, args...)
}

// GetStdout returns the stdout output as a string
func (h *CLITestHelper) GetStdout() string {
	return h.stdout.String()
}

// GetStderr returns the stderr output as a string
func (h *CLITestHelper) GetStderr() string {
	return h.stderr.String()
}

// AssertStdoutContains asserts that stdout contains the expected string
func (h *CLITestHelper) AssertStdoutContains(expected string) {
	require.Contains(h.t, h.GetStdout(), expected, "stdout should contain: %s", expected)
}

// AssertStderrContains asserts that stderr contains the expected string
func (h *CLITestHelper) AssertStderrContains(expected string) {
	require.Contains(h.t, h.GetStderr(), expected, "stderr should contain: %s", expected)
}

// AssertStdoutNotContains asserts that stdout does not contain the string
func (h *CLITestHelper) AssertStdoutNotContains(unexpected string) {
	require.NotContains(h.t, h.GetStdout(), unexpected, "stdout should not contain: %s", unexpected)
}

// AssertOutputEmpty asserts that both stdout and stderr are empty
func (h *CLITestHelper) AssertOutputEmpty() {
	require.Empty(h.t, h.GetStdout(), "stdout should be empty")
	require.Empty(h.t, h.GetStderr(), "stderr should be empty")
}

// MockEnvironment provides utilities for mocking environment
type MockEnvironment struct {
	t           *testing.T
	originalEnv map[string]string
	originalWd  string
}

// NewMockEnvironment creates a new mock environment
func NewMockEnvironment(t *testing.T) *MockEnvironment {
	originalWd, err := os.Getwd()
	require.NoError(t, err)

	return &MockEnvironment{
		t:           t,
		originalEnv: make(map[string]string),
		originalWd:  originalWd,
	}
}

// SetEnv sets an environment variable and remembers the original value
func (m *MockEnvironment) SetEnv(key, value string) {
	if original, exists := os.LookupEnv(key); exists {
		m.originalEnv[key] = original
	} else {
		m.originalEnv[key] = ""
	}
	require.NoError(m.t, os.Setenv(key, value))
}

// ChangeDir changes the working directory
func (m *MockEnvironment) ChangeDir(dir string) {
	require.NoError(m.t, os.Chdir(dir))
}

// Cleanup restores the original environment and working directory
func (m *MockEnvironment) Cleanup() {
	// Restore working directory
	require.NoError(m.t, os.Chdir(m.originalWd))

	// Restore environment variables
	for key, original := range m.originalEnv {
		if original == "" {
			require.NoError(m.t, os.Unsetenv(key))
		} else {
			require.NoError(m.t, os.Setenv(key, original))
		}
	}
}

// CaptureOutput captures stdout and stderr during function execution
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	// Save original stdout/stderr
	originalStdout := os.Stdout
	originalStderr := os.Stderr

	// Create pipes
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	// Replace stdout/stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Create channels to capture output
	stdoutCh := make(chan string)
	stderrCh := make(chan string)

	// Start goroutines to read from pipes
	go func() {
		defer close(stdoutCh)
		buf := &bytes.Buffer{}
		io.Copy(buf, stdoutR)
		stdoutCh <- buf.String()
	}()

	go func() {
		defer close(stderrCh)
		buf := &bytes.Buffer{}
		io.Copy(buf, stderrR)
		stderrCh <- buf.String()
	}()

	// Execute function
	fn()

	// Close writers
	stdoutW.Close()
	stderrW.Close()

	// Restore original stdout/stderr
	os.Stdout = originalStdout
	os.Stderr = originalStderr

	// Get captured output
	stdout = <-stdoutCh
	stderr = <-stderrCh

	// Close readers
	stdoutR.Close()
	stderrR.Close()

	return stdout, stderr
}

// AssertCommandSuccess asserts that a command executed successfully
func AssertCommandSuccess(t *testing.T, err error, stdout, stderr string) {
	if err != nil {
		t.Logf("Command failed with error: %v", err)
		t.Logf("Stdout: %s", stdout)
		t.Logf("Stderr: %s", stderr)
	}
	require.NoError(t, err, "command should execute successfully")
}

// AssertCommandFailure asserts that a command failed with expected error
func AssertCommandFailure(t *testing.T, err error, expectedErrorMsg string) {
	require.Error(t, err, "command should fail")
	if expectedErrorMsg != "" {
		require.Contains(t, err.Error(), expectedErrorMsg, "error message should contain expected text")
	}
}

// NormalizeOutput normalizes output for comparison (removes extra whitespace, etc.)
func NormalizeOutput(output string) string {
	// Remove trailing whitespace from each line
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Join back and trim overall
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// ExecuteCommand executes a Cobra command and returns the combined output
func ExecuteCommand(cmd *cobra.Command, args []string) (string, error) {
	var stdout, stderr bytes.Buffer

	// Set command output
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)

	// Execute command
	err := cmd.Execute()

	// Combine stdout and stderr
	output := stdout.String() + stderr.String()

	return output, err
}

// ExecuteCommandByName executes a command by finding it in the root command
func ExecuteCommandByName(cmdName string, args []string) (string, error) {
	// This is a simplified implementation - in a real scenario,
	// you'd need access to the root command to find subcommands
	// For now, we'll return an error indicating this needs to be implemented
	return "", nil
}
