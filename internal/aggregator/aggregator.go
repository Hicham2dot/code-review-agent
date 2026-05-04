package aggregator

import (
	"code-review-agent/internal/models"
	"crypto/sha256"
	"fmt"
	"sort"
	"time"
)

func Aggregate(localIssues, llmIssues []models.Issue, hunks []models.DiffHunk, diff string) models.AnalysisResult {
	startTime := time.Now()

	// 1. Merge
	allIssues := append(localIssues, llmIssues...)

	// 2. Deduplicate (keep highest confidence per file+line+type)
	deduped := deduplicate(allIssues)

	// 3. Sort by severity (critical → major → minor), then by file+line
	sorted := sortByPriority(deduped)

	// 4. Calculate summary and build result
	summary := calculateSummary(sorted)
	result := models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   hashDiff(diff),
		FileCount:  countFiles(hunks),
		TotalLines: countTotalLines(hunks),
		Issues:     sorted,
		Summary:    summary,
		Duration:   float64(time.Since(startTime).Milliseconds()),
	}

	return result
}

func deduplicate(issues []models.Issue) []models.Issue {
	if len(issues) == 0 {
		return []models.Issue{}
	}

	// Key: file + start_line + type
	seen := make(map[string]*models.Issue)

	for i := range issues {
		key := fmt.Sprintf("%s:%d:%s", issues[i].Location.File, issues[i].Location.StartLine, issues[i].Type)

		if existing, found := seen[key]; found {
			// Keep the one with higher confidence
			if issues[i].Confidence > existing.Confidence {
				seen[key] = &issues[i]
			}
		} else {
			seen[key] = &issues[i]
		}
	}

	// Extract deduplicated issues
	result := make([]models.Issue, 0, len(seen))
	for _, issue := range seen {
		result = append(result, *issue)
	}

	return result
}

func sortByPriority(issues []models.Issue) []models.Issue {
	// Create a copy to avoid modifying input
	sorted := make([]models.Issue, len(issues))
	copy(sorted, issues)

	sort.Slice(sorted, func(i, j int) bool {
		// Priority: critical > major > minor
		severityOrder := map[string]int{"critical": 0, "major": 1, "minor": 2}
		iSev := severityOrder[sorted[i].Severity]
		jSev := severityOrder[sorted[j].Severity]

		if iSev != jSev {
			return iSev < jSev
		}

		// Same severity: sort by file, then by line number
		if sorted[i].Location.File != sorted[j].Location.File {
			return sorted[i].Location.File < sorted[j].Location.File
		}

		return sorted[i].Location.StartLine < sorted[j].Location.StartLine
	})
	return sorted
}

func calculateSummary(issues []models.Issue) models.Summary {
	summary := models.Summary{
		CriticalCount: 0,
		MajorCount:    0,
		MinorCount:    0,
		TotalIssues:   len(issues),
		Quality:       "A",
		Confidence:    0.0,
	}

	if len(issues) == 0 {
		summary.Quality = "A"
		summary.Confidence = 1.0
		return summary
	}

	var totalConfidence float64

	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			summary.CriticalCount++
		case "major":
			summary.MajorCount++
		case "minor":
			summary.MinorCount++
		}
		totalConfidence += issue.Confidence
	}

	// Average confidence
	summary.Confidence = totalConfidence / float64(len(issues))

	// Quality score: 100 - (critical*10 + major*5 + minor*1)
	score := 100 - (summary.CriticalCount*10 + summary.MajorCount*5 + summary.MinorCount*1)
	if score < 0 {
		score = 0
	}

	// Letter grade
	switch {
	case score >= 90:
		summary.Quality = "A"
	case score >= 75:
		summary.Quality = "B"
	case score >= 60:
		summary.Quality = "C"
	default:
		summary.Quality = "D"
	}

	return summary
}

func hashDiff(diff string) string {
	hash := sha256.Sum256([]byte(diff))
	return fmt.Sprintf("%x", hash)[:16]
}

func countFiles(hunks []models.DiffHunk) int {
	if len(hunks) == 0 {
		return 0
	}

	files := make(map[string]bool)
	for _, hunk := range hunks {
		files[hunk.File] = true
	}

	return len(files)
}

func countTotalLines(hunks []models.DiffHunk) int {
	total := 0
	for _, hunk := range hunks {
		total += len(hunk.AddedLines) + len(hunk.RemovedLines)
	}
	return total
}
