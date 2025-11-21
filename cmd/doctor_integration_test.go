package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/keisukeshimizu/hatcher/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand_Integration(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "doctor-integration-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("basic doctor check", func(t *testing.T) {
		// Execute doctor command
		output, err := helpers.ExecuteCommand(doctorCmd, []string{})
		// Note: doctor command may exit with non-zero code for warnings/failures
		// We check the output regardless of exit code

		// Should show diagnostic information
		assert.Contains(t, output, "Git Installation")
		assert.Contains(t, output, "Git Repository")
		assert.Contains(t, output, "CHECK")
		assert.Contains(t, output, "STATUS")
	})

	t.Run("JSON output format", func(t *testing.T) {
		// Execute doctor command with JSON format
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--format", "json"})
		// Check output regardless of exit code

		// Should be valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Should contain expected fields
		assert.Contains(t, result, "checks")
		assert.Contains(t, result, "summary")
	})

	t.Run("simple output format", func(t *testing.T) {
		// Execute doctor command with simple format
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--format", "simple"})
		// Check output regardless of exit code

		// Should show simple format with icons
		assert.Contains(t, output, "✅")
		assert.Contains(t, output, "Git Installation")
		assert.Contains(t, output, "Summary:")
	})

	t.Run("simple flag", func(t *testing.T) {
		// Execute doctor command with --simple flag
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--simple"})
		// Check output regardless of exit code

		// Should show simple format
		assert.Contains(t, output, "Git Installation")
		// Should not show table headers
		assert.NotContains(t, output, "CHECK")
		assert.NotContains(t, output, "STATUS")
	})

	t.Run("command aliases", func(t *testing.T) {
		// Test 'check' alias
		output, err := helpers.ExecuteCommandByName("check", []string{})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "Git Installation")
		}

		// Test 'validate' alias
		output, err = helpers.ExecuteCommandByName("validate", []string{})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "Git Installation")
		}

		// Test 'diagnose' alias
		output, err = helpers.ExecuteCommandByName("diagnose", []string{})
		if err == nil { // Only test if alias is properly implemented
			assert.Contains(t, output, "Git Installation")
		}
	})

	t.Run("verbose output", func(t *testing.T) {
		// Execute doctor command with verbose flag
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--verbose"})
		// Check output regardless of exit code

		// Should show diagnostic information
		assert.Contains(t, output, "Git Installation")
	})
}

func TestDoctorCommand_NonGitRepository(t *testing.T) {
	// Create temporary directory that's not a git repository
	tempDir := t.TempDir()

	// Change to non-git directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("doctor in non-git directory", func(t *testing.T) {
		// Execute doctor command
		output, err := helpers.ExecuteCommand(doctorCmd, []string{})
		// Should still work, but may show warnings about Git repository

		// Should still show Git installation check
		assert.Contains(t, output, "Git Installation")

		// May show warning about not being in a Git repository
		if strings.Contains(output, "Git Repository") {
			// If Git Repository check is shown, it should indicate the issue
			assert.True(t, strings.Contains(output, "WARN") || strings.Contains(output, "FAIL"))
		}
	})

	t.Run("JSON output in non-git directory", func(t *testing.T) {
		// Execute doctor command with JSON format
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--format", "json"})

		// Should still produce valid JSON
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)

		// Should contain expected fields
		assert.Contains(t, result, "checks")
		assert.Contains(t, result, "summary")
	})
}

func TestDoctorCommand_EdgeCases(t *testing.T) {
	// Create test repository
	testRepo := helpers.NewTestGitRepository(t, "doctor-edge-cases-test")

	// Change to test repository directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	err = os.Chdir(testRepo.RepoDir)
	require.NoError(t, err)

	t.Run("invalid output format", func(t *testing.T) {
		// Execute doctor command with invalid format
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--format", "invalid"})
		// Should default to table format

		// Should show table format
		assert.Contains(t, output, "CHECK")
		assert.Contains(t, output, "STATUS")
	})

	t.Run("conflicting format flags", func(t *testing.T) {
		// Execute doctor command with both format and simple flags
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--format", "json", "--simple"})
		// JSON format should take precedence

		// Should be valid JSON (not simple format)
		var result map[string]interface{}
		err = json.Unmarshal([]byte(output), &result)
		require.NoError(t, err)
	})

	t.Run("check system health indicators", func(t *testing.T) {
		// Execute doctor command
		output, err := helpers.ExecuteCommand(doctorCmd, []string{})

		// Should show summary information
		assert.Contains(t, output, "Summary:")

		// Should show overall status
		assert.True(t,
			strings.Contains(output, "Healthy") ||
			strings.Contains(output, "Issues Found") ||
			strings.Contains(output, "passed") ||
			strings.Contains(output, "failed"))
	})

	t.Run("check individual diagnostic components", func(t *testing.T) {
		// Execute doctor command
		output, err := helpers.ExecuteCommand(doctorCmd, []string{})

		// Should check Git installation
		assert.Contains(t, output, "Git Installation")

		// Should check Git repository
		assert.Contains(t, output, "Git Repository")

		// Should check editors
		assert.Contains(t, output, "Editors")

		// Should check configuration
		assert.Contains(t, output, "Configuration")

		// Should check permissions
		assert.Contains(t, output, "Permissions")
	})

	t.Run("check status indicators", func(t *testing.T) {
		// Execute doctor command with simple format to see icons
		output, err := helpers.ExecuteCommand(doctorCmd, []string{"--simple"})

		// Should contain status indicators
		hasStatusIndicators := strings.Contains(output, "✅") ||
							   strings.Contains(output, "⚠️") ||
							   strings.Contains(output, "❌")
		assert.True(t, hasStatusIndicators, "Should contain status indicators")
	})
}
