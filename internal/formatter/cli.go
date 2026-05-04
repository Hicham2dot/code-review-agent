package formatter

import (
	"code-review-agent/internal/models"
	"fmt"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorBlue   = "\033[34m"
)

func FormatCLI(result *models.AnalysisResult) string {
	output := ""

	output += fmt.Sprintf("%s=== Code Review Analysis ===%s\n", colorBlue, colorReset)
	output += fmt.Sprintf("Timestamp: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	output += fmt.Sprintf("Files: %d | Total lines: %d | Duration: %.2f ms\n\n", result.FileCount, result.TotalLines, result.Duration)

	output += fmt.Sprintf("Quality: %s | Confidence: %.2f\n", severityColor(result.Summary.Quality, result.Summary.Quality), result.Summary.Confidence)
	output += fmt.Sprintf("Issues: %d (Critical: %d, Major: %d, Minor: %d)\n\n",
		result.Summary.TotalIssues, result.Summary.CriticalCount, result.Summary.MajorCount, result.Summary.MinorCount)

	if len(result.Issues) == 0 {
		output += fmt.Sprintf("%s✓ No issues found%s\n", colorGreen, colorReset)
	} else {
		output += fmt.Sprintf("%s=== Issues ===%s\n", colorBlue, colorReset)
		for i, issue := range result.Issues {
			color := severityColor(issue.Severity, issue.Severity)
			badge := severityBadge(issue.Severity)
			output += fmt.Sprintf("%d. %s%s %s%s [%s:%d]\n", i+1, color, badge, issue.Type, colorReset, issue.Location.File, issue.Location.StartLine)
			output += fmt.Sprintf("   Message: %s\n", issue.Message)
			if issue.Suggestion != "" {
				output += fmt.Sprintf("   Suggestion: %s\n", issue.Suggestion)
			}
			output += fmt.Sprintf("   Confidence: %.2f | Source: %s\n\n", issue.Confidence, issue.Source)
		}
	}

	return output
}

func severityColor(severity, text string) string {
	switch severity {
	case "critical":
		return fmt.Sprintf("%s%s%s", colorRed, text, colorReset)
	case "major":
		return fmt.Sprintf("%s%s%s", colorYellow, text, colorReset)
	case "minor":
		return fmt.Sprintf("%s%s%s", colorGreen, text, colorReset)
	default:
		return text
	}
}

func severityBadge(severity string) string {
	switch severity {
	case "critical":
		return "🔴"
	case "major":
		return "🟡"
	case "minor":
		return "🟢"
	default:
		return "⚪"
	}
}
