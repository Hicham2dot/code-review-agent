package formatter

import (
	"code-review-agent/internal/models"
	"fmt"
	"strings"
)

func FormatMarkdown(result *models.AnalysisResult) string {
	var sb strings.Builder

	sb.WriteString("# Code Review Report\n\n")
	sb.WriteString(fmt.Sprintf("**Timestamp**: %s  \n", result.Timestamp.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("**Files Modified**: %d | **Total Lines Changed**: %d  \n", result.FileCount, result.TotalLines))
	sb.WriteString(fmt.Sprintf("**Analysis Duration**: %.2f ms\n\n", result.Duration))

	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Quality Grade | **%s** |\n", result.Summary.Quality))
	sb.WriteString(fmt.Sprintf("| Total Issues | %d |\n", result.Summary.TotalIssues))
	sb.WriteString(fmt.Sprintf("| Critical | %d |\n", result.Summary.CriticalCount))
	sb.WriteString(fmt.Sprintf("| Major | %d |\n", result.Summary.MajorCount))
	sb.WriteString(fmt.Sprintf("| Minor | %d |\n", result.Summary.MinorCount))
	sb.WriteString(fmt.Sprintf("| Avg Confidence | %.2f |\n\n", result.Summary.Confidence))

	if len(result.Issues) == 0 {
		sb.WriteString("✅ **No issues detected!**\n")
	} else {
		sb.WriteString("## Issues\n\n")
		sb.WriteString("| # | Type | Severity | File | Line | Message | Confidence |\n")
		sb.WriteString("|---|------|----------|------|------|---------|------------|\n")

		for i, issue := range result.Issues {
			badge := severityMDBadge(issue.Severity)
			sb.WriteString(fmt.Sprintf("| %d | `%s` | %s | `%s` | %d | %s | %.2f |\n",
				i+1,
				issue.Type,
				badge,
				issue.Location.File,
				issue.Location.StartLine,
				issue.Message,
				issue.Confidence,
			))
		}

		sb.WriteString("\n## Details\n\n")
		for i, issue := range result.Issues {
			sb.WriteString(fmt.Sprintf("### %d. %s (%s)\n\n", i+1, issue.Type, issue.Severity))
			sb.WriteString(fmt.Sprintf("**Location**: `%s:%d`  \n", issue.Location.File, issue.Location.StartLine))
			sb.WriteString(fmt.Sprintf("**Source**: %s  \n", issue.Source))
			sb.WriteString(fmt.Sprintf("**Confidence**: %.2f\n\n", issue.Confidence))
			sb.WriteString(fmt.Sprintf("**Message**: %s\n\n", issue.Message))
			if issue.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("**Suggestion**: %s\n\n", issue.Suggestion))
			}
		}
	}

	return sb.String()
}

func severityMDBadge(severity string) string {
	switch severity {
	case "critical":
		return "🔴 **Critical**"
	case "major":
		return "🟡 **Major**"
	case "minor":
		return "🟢 **Minor**"
	default:
		return "⚪ Unknown"
	}
}
