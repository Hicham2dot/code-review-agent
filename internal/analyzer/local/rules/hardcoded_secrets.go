package rules

import (
	"code-review-agent/internal/models"
	"fmt"
	"regexp"
)

var secretPatterns = map[string]*regexp.Regexp{
	"api_key":      regexp.MustCompile(`(?i)api[_-]?key\s*=\s*["'][a-zA-Z0-9\-_.]{10,}`),
	"password":     regexp.MustCompile(`(?i)password\s*=\s*["'][^"']{6,}`),
	"token":        regexp.MustCompile(`(?i)token\s*=\s*["'][a-zA-Z0-9\-_.]{10,}`),
	"secret":       regexp.MustCompile(`(?i)secret\s*=\s*["'][^"']{8,}`),
	"db_password":  regexp.MustCompile(`(?i)db[_-]password\s*=\s*["'][^"']{6,}`),
	"aws_key":      regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
	"private_key":  regexp.MustCompile(`-----BEGIN (RSA|DSA|EC|PGP) PRIVATE KEY`),
}

func checkHardcodedSecrets(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	for _, hunk := range hunks {
		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i

			for secretType, pattern := range secretPatterns {
				if pattern.MatchString(line) {
					issues = append(issues, models.Issue{
						ID:       fmt.Sprintf("secret-%s-%d", secretType, lineNum),
						Type:     "hardcoded_secrets",
						Severity: "critical",
						Location: models.Location{
							File:      hunk.File,
							StartLine: lineNum,
							EndLine:   lineNum,
						},
						Message:    fmt.Sprintf("Hardcoded %s detected in code", secretType),
						Suggestion: "Use environment variables or secret management service",
						Confidence: 0.98,
						Source:     "local_analyzer",
					})
					break
				}
			}
		}
	}

	return issues
}
