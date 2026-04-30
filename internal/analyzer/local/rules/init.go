package rules

import (
	"code-review-agent/internal/models"
)

// CheckDeprecatedFunction wraps the deprecated function checker
func CheckDeprecatedFunction(hunks []models.DiffHunk) []models.Issue {
	return checkDeprecatedFunction(hunks)
}

// CheckHardcodedSecrets wraps the hardcoded secrets checker
func CheckHardcodedSecrets(hunks []models.DiffHunk) []models.Issue {
	return checkHardcodedSecrets(hunks)
}

// CheckSQLInjection wraps the SQL injection checker
func CheckSQLInjection(hunks []models.DiffHunk) []models.Issue {
	return checkSQLInjection(hunks)
}

// CheckTodoComment wraps the todo comment checker
func CheckTodoComment(hunks []models.DiffHunk) []models.Issue {
	return checkTodoComment(hunks)
}

// CheckLargeFunction wraps the large function checker
func CheckLargeFunction(hunks []models.DiffHunk) []models.Issue {
	return checkLargeFunction(hunks)
}

// CheckMissingErrorHandling wraps the missing error handling checker
func CheckMissingErrorHandling(hunks []models.DiffHunk) []models.Issue {
	return checkMissingErrorHandling(hunks)
}
