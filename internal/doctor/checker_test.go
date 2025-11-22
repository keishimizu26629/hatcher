package doctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/keisukeshimizu/hatcher/internal/git"
	"github.com/keisukeshimizu/hatcher/test/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChecker_CheckSystem(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "doctor-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	checker := NewChecker(repo)

	t.Run("check healthy system", func(t *testing.T) {
		// Run system check
		result, err := checker.CheckSystem()
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have basic checks
		assert.NotEmpty(t, result.Checks)

		// Find Git check
		var gitCheck *CheckResult
		for i := range result.Checks {
			if result.Checks[i].Name == "Git Installation" {
				gitCheck = &result.Checks[i]
				break
			}
		}
		require.NotNil(t, gitCheck, "Git check should be present")
		assert.Equal(t, CheckStatusPass, gitCheck.Status)
	})

	t.Run("check with detailed output", func(t *testing.T) {
		// Run system check with details
		result, err := checker.CheckSystem()
		require.NoError(t, err)

		// Should have detailed information
		for _, check := range result.Checks {
			assert.NotEmpty(t, check.Name)
			assert.NotEmpty(t, check.Description)
		}
	})
}

func TestChecker_CheckGitInstallation(t *testing.T) {
	checker := NewChecker(nil) // No repo needed for this test

	t.Run("check Git installation", func(t *testing.T) {
		// Check Git installation
		result := checker.CheckGitInstallation()
		assert.NotNil(t, result)
		assert.Equal(t, "Git Installation", result.Name)

		// Should pass if Git is available
		if result.Status == CheckStatusPass {
			assert.NotEmpty(t, result.Details)
			assert.Contains(t, result.Details, "version")
		}
	})
}

func TestChecker_CheckGitRepository(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "git-repo-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	checker := NewChecker(repo)

	t.Run("check valid Git repository", func(t *testing.T) {
		// Check Git repository
		result := checker.CheckGitRepository()
		assert.NotNil(t, result)
		assert.Equal(t, "Git Repository", result.Name)
		assert.Equal(t, CheckStatusPass, result.Status)
		assert.Contains(t, result.Details, testRepo.RepoDir)
	})

	t.Run("check repository with worktrees", func(t *testing.T) {
		// Create a worktree
		branchName := "feature/doctor-test"
		worktreePath := filepath.Join(testRepo.TempDir, "git-repo-test-feature-doctor-test")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Check Git repository
		result := checker.CheckGitRepository()
		assert.Equal(t, CheckStatusPass, result.Status)
		assert.Contains(t, result.Details, "worktrees")
	})
}

func TestChecker_CheckWorktrees(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "worktrees-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	checker := NewChecker(repo)

	t.Run("check worktrees with no issues", func(t *testing.T) {
		// Create clean worktrees
		branchName1 := "feature/clean-1"
		branchName2 := "feature/clean-2"
		worktreePath1 := filepath.Join(testRepo.TempDir, "worktrees-test-feature-clean-1")
		worktreePath2 := filepath.Join(testRepo.TempDir, "worktrees-test-feature-clean-2")

		err := repo.CreateWorktree(worktreePath1, branchName1, true)
		require.NoError(t, err)
		err = repo.CreateWorktree(worktreePath2, branchName2, true)
		require.NoError(t, err)

		// Check worktrees
		result := checker.CheckWorktrees()
		assert.NotNil(t, result)
		assert.Equal(t, "Worktrees", result.Name)
		assert.Equal(t, CheckStatusPass, result.Status)
		assert.Contains(t, result.Details, "2 worktrees")
	})

	t.Run("check worktrees with missing directories", func(t *testing.T) {
		// Create a worktree then remove its directory
		branchName := "feature/missing-dir"
		worktreePath := filepath.Join(testRepo.TempDir, "worktrees-test-feature-missing-dir")

		err := repo.CreateWorktree(worktreePath, branchName, true)
		require.NoError(t, err)

		// Remove the directory
		err = os.RemoveAll(worktreePath)
		require.NoError(t, err)

		// Check worktrees
		result := checker.CheckWorktrees()
		assert.Equal(t, CheckStatusWarn, result.Status)
		assert.Contains(t, result.Details, "missing")
	})
}

func TestChecker_CheckEditors(t *testing.T) {
	checker := NewChecker(nil) // No repo needed for this test

	t.Run("check available editors", func(t *testing.T) {
		// Check editors
		result := checker.CheckEditors()
		assert.NotNil(t, result)
		assert.Equal(t, "Editors", result.Name)

		// Should report available editors
		assert.NotEmpty(t, result.Details)
	})
}

func TestChecker_CheckConfiguration(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "config-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	checker := NewChecker(repo)

	t.Run("check configuration files", func(t *testing.T) {
		// Check configuration
		result := checker.CheckConfiguration()
		assert.NotNil(t, result)
		assert.Equal(t, "Configuration", result.Name)

		// Should check for auto-copy config
		assert.Contains(t, result.Details, "auto-copy")
	})

	t.Run("check with auto-copy config", func(t *testing.T) {
		// Create auto-copy config file
		configPath := filepath.Join(testRepo.RepoDir, ".hatcher-auto-copy.json")
		configContent := `{
			"version": 2,
			"items": [
				{
					"path": ".ai/",
					"directory": true,
					"recursive": true
				}
			]
		}`
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Check configuration
		result := checker.CheckConfiguration()
		assert.Equal(t, CheckStatusPass, result.Status)
		assert.Contains(t, result.Details, "found")
	})
}

func TestChecker_CheckPermissions(t *testing.T) {
	// Create test repository
	testRepo := testutil.NewTestGitRepository(t, "permissions-test")
	repo, err := git.NewRepositoryFromPath(testRepo.RepoDir)
	require.NoError(t, err)

	checker := NewChecker(repo)

	t.Run("check file permissions", func(t *testing.T) {
		// Check permissions
		result := checker.CheckPermissions()
		assert.NotNil(t, result)
		assert.Equal(t, "Permissions", result.Name)

		// Should check repository access
		assert.NotEmpty(t, result.Details)
	})
}

func TestDiagnosticResult_FormatOutput(t *testing.T) {
	// Create sample diagnostic result
	result := &DiagnosticResult{
		Checks: []CheckResult{
			{
				Name:        "Test Check 1",
				Description: "This is a test check",
				Status:      CheckStatusPass,
				Details:     "Everything is working fine",
			},
			{
				Name:        "Test Check 2",
				Description: "This is another test check",
				Status:      CheckStatusWarn,
				Details:     "Minor issue detected",
				Suggestions: []string{"Fix the minor issue"},
			},
			{
				Name:        "Test Check 3",
				Description: "This is a failing check",
				Status:      CheckStatusFail,
				Details:     "Critical error found",
				Suggestions: []string{"Fix the critical error", "Contact support"},
			},
		},
		Summary: DiagnosticSummary{
			Total:   3,
			Passed:  1,
			Warned:  1,
			Failed:  1,
			Healthy: false,
		},
	}

	t.Run("format as table", func(t *testing.T) {
		output := result.FormatAsTable()
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "Test Check 1")
		assert.Contains(t, output, "Test Check 2")
		assert.Contains(t, output, "Test Check 3")
		assert.Contains(t, output, "PASS")
		assert.Contains(t, output, "WARN")
		assert.Contains(t, output, "FAIL")
	})

	t.Run("format as JSON", func(t *testing.T) {
		output := result.FormatAsJSON()
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "Test Check 1")
		assert.Contains(t, output, "\"status\":")
		assert.Contains(t, output, "\"summary\":")
	})

	t.Run("format as simple", func(t *testing.T) {
		output := result.FormatAsSimple()
		assert.NotEmpty(t, output)
		assert.Contains(t, output, "✅")
		assert.Contains(t, output, "⚠️")
		assert.Contains(t, output, "❌")
	})
}

func TestDiagnosticResult_GetOverallStatus(t *testing.T) {
	t.Run("all checks pass", func(t *testing.T) {
		result := &DiagnosticResult{
			Checks: []CheckResult{
				{Status: CheckStatusPass},
				{Status: CheckStatusPass},
			},
		}

		status := result.GetOverallStatus()
		assert.Equal(t, CheckStatusPass, status)
	})

	t.Run("some checks warn", func(t *testing.T) {
		result := &DiagnosticResult{
			Checks: []CheckResult{
				{Status: CheckStatusPass},
				{Status: CheckStatusWarn},
			},
		}

		status := result.GetOverallStatus()
		assert.Equal(t, CheckStatusWarn, status)
	})

	t.Run("some checks fail", func(t *testing.T) {
		result := &DiagnosticResult{
			Checks: []CheckResult{
				{Status: CheckStatusPass},
				{Status: CheckStatusWarn},
				{Status: CheckStatusFail},
			},
		}

		status := result.GetOverallStatus()
		assert.Equal(t, CheckStatusFail, status)
	})
}
