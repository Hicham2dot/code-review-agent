package rules

import (
	"code-review-agent/internal/models"
	"fmt"
	"regexp"
	"strings"
)

var sqlInjectionPatterns = map[string]*regexp.Regexp{
	"string_concat":           regexp.MustCompile(`(?i)SELECT.*\+\s*\w+`),
	"fmt_printf":              regexp.MustCompile(`(?i)fmt\.Sprintf.*SELECT.*%s`),
	"string_formatting":       regexp.MustCompile(`(?i)SELECT.*\$\{.*\}`),
	"unparameterized_queries": regexp.MustCompile(`(?i)SELECT.*WHERE.*=.*"\s*\+|=.*fmt\.Sprintf`),
}

var safeQueryIndicators = []string{"?", "$1", "$2", "$3", "@param", "prepared"}

func checkSQLInjection(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	for _, hunk := range hunks {
		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i

			// Check for vulnerable patterns
			for patternName, pattern := range sqlInjectionPatterns {
				if pattern.MatchString(line) {
					// Check if query looks safe
					isSafe := false
					for _, indicator := range safeQueryIndicators {
						if strings.Contains(line, indicator) {
							isSafe = true
							break
						}
					}

					if !isSafe {
						issues = append(issues, models.Issue{
							ID:       fmt.Sprintf("sql-inj-%d", lineNum),
							Type:     "sql_injection",
							Severity: "critical",
							Location: models.Location{
								File:      hunk.File,
								StartLine: lineNum,
								EndLine:   lineNum,
							},
							Message:    fmt.Sprintf("Potential SQL injection vulnerability detected (%s)", patternName),
							Suggestion: "Use parameterized queries or prepared statements",
							Confidence: 0.85,
							Source:     "local_analyzer",
						})
						break
					}
				}
			}
		}
	}

	return issues
}
