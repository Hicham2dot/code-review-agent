package rules

import (
	"code-review-agent/internal/models"
	"fmt"
	"regexp"
)

var deprecatedFunctions = map[string]string{
	"io.ReadAll":        "Use io.ReadAll directly (already available)",
	"ioutil.ReadAll":    "Use io.ReadAll instead (ioutil is deprecated)",
	"ioutil.ReadFile":   "Use os.ReadFile instead",
	"ioutil.WriteFile":  "Use os.WriteFile instead",
	"ioutil.TempDir":    "Use os.MkdirTemp instead",
	"ioutil.TempFile":   "Use os.CreateTemp instead",
	"sql.ErrNoRows":     "Check if rows.Next() returned false",
	"strings.Title":     "Use cases.Title instead (Title is deprecated)",
	"math.Floor":        "Use math.Floor with proper type handling",
	"time.Now.Unix":     "Use time.Now().Unix() properly",
}

func checkDeprecatedFunction(hunks []models.DiffHunk) []models.Issue {
	var issues []models.Issue

	for _, hunk := range hunks {
		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i

			for deprecated, replacement := range deprecatedFunctions {
				pattern := regexp.MustCompile(regexp.QuoteMeta(deprecated) + `\s*\(`)
				if pattern.MatchString(line) {
					issues = append(issues, models.Issue{
						ID:       fmt.Sprintf("deprecated-func-%d", lineNum),
						Type:     "deprecated_function",
						Severity: "minor",
						Location: models.Location{
							File:      hunk.File,
							StartLine: lineNum,
							EndLine:   lineNum,
						},
						Message:    fmt.Sprintf("Deprecated function '%s' detected", deprecated),
						Suggestion: fmt.Sprintf("Replace with: %s", replacement),
						Confidence: 0.92,
						Source:     "local_analyzer",
					})
					break
				}
			}
		}
	}

	return issues
}
