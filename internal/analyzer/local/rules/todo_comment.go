package rules

import (
	"code-review-agent/internal/models"
	"fmt"
	"regexp"
)

var todoPattern = regexp.MustCompile(`(?i)(TODO|FIXME|XXX|HACK|BUG):\s*(.+)`)

func checkTodoComment(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	for _, hunk := range hunks {
		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i

			if matches := todoPattern.FindStringSubmatch(line); matches != nil {
				issues = append(issues, models.Issue{
					ID:       fmt.Sprintf("todo-%d", lineNum),
					Type:     "todo_comment",
					Severity: "minor",
					Location: models.Location{
						File:      hunk.File,
						StartLine: lineNum,
						EndLine:   lineNum,
					},
					Message:    fmt.Sprintf("Found %s comment: %s", matches[1], matches[2]),
					Suggestion: "Address this TODO before merging",
					Confidence: 0.99,
					Source:     "local_analyzer",
				})
			}
		}
	}

	return issues
}
