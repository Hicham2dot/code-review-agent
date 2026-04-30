package rules

import (
	"code-review-agent/internal/models"
	"regexp"
	"strings"
)

func checkMissingErrorHandling(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	// Patterns that indicate a function might return an error
	errorReturnPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\)\s*\(\w+.*error\s*\)`),
		regexp.MustCompile(`\)\s*error`),
		regexp.MustCompile(`\),\s*error\)`),
	}

	for _, hunk := range hunks {
		for _, line := range hunk.AddedLines {
			// Skip lines that already handle errors
			if strings.Contains(line, "if err != nil") {
				continue
			}
			if strings.Contains(line, "_ =") {
				continue
			}
			if strings.Contains(line, "defer") {
				continue
			}
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}

			// Check if line is a function definition with error return
			for _, pattern := range errorReturnPatterns {
				if pattern.MatchString(line) {
					// This is a function signature, not a call that needs error handling
					// So we skip this check
					continue
				}
			}
		}
	}

	return issues
}
