package rules

import (
	"code-review-agent/internal/models"
	"fmt"
	"strings"
)

const functionSizeThreshold = 50

func checkLargeFunction(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	for _, hunk := range hunks {
		// Track function boundaries
		var currentFunction string
		var functionStartLine int
		var functionLineCount int
		var braceDepth int

		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i
			trimmed := strings.TrimSpace(line)

			// Detect function start
			if strings.Contains(trimmed, "func ") && strings.Contains(trimmed, "{") {
				currentFunction = strings.TrimSpace(strings.Split(trimmed, "{")[0])
				functionStartLine = lineNum
				functionLineCount = 1
				braceDepth = 1
				continue
			}

			// Count lines within function
			if currentFunction != "" {
				functionLineCount++
				braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")

				// Detect function end
				if braceDepth == 0 {
					if functionLineCount > functionSizeThreshold {
						issues = append(issues, models.Issue{
							ID:       fmt.Sprintf("large-func-%d", functionStartLine),
							Type:     "large_function",
							Severity: "major",
							Location: models.Location{
								File:      hunk.File,
								StartLine: functionStartLine,
								EndLine:   lineNum,
							},
							Message:    fmt.Sprintf("Function %s is too large (%d lines)", currentFunction, functionLineCount),
							Suggestion: "Consider breaking this function into smaller, testable units",
							Confidence: 0.95,
							Source:     "local_analyzer",
						})
					}
					currentFunction = ""
					functionLineCount = 0
				}
			}
		}
	}

	return issues
}
