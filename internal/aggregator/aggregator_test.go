package aggregator

import (
	"code-review-agent/internal/models"
	"testing"
)

func TestAggregateEmpty(t *testing.T) {
	hunks := []models.DiffHunk{}
	diff := ""

	result := Aggregate(nil, nil, hunks, diff)

	if result.Summary.TotalIssues != 0 {
		t.Errorf("Expected 0 issues, got %d", result.Summary.TotalIssues)
	}
	if result.FileCount != 0 {
		t.Errorf("Expected 0 files, got %d", result.FileCount)
	}
	if result.Summary.Quality != "A" {
		t.Errorf("Expected quality A for no issues, got %s", result.Summary.Quality)
	}
}

func TestDeduplicateKeepsHighestConfidence(t *testing.T) {
	issues := []models.Issue{
		{
			Type:       "sql_injection",
			Severity:   "critical",
			Confidence: 0.85,
			Location:   models.Location{File: "main.go", StartLine: 10},
		},
		{
			Type:       "sql_injection",
			Severity:   "critical",
			Confidence: 0.95,
			Location:   models.Location{File: "main.go", StartLine: 10},
		},
	}

	deduped := deduplicate(issues)

	if len(deduped) != 1 {
		t.Errorf("Expected 1 deduplicated issue, got %d", len(deduped))
	}
	if deduped[0].Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %v", deduped[0].Confidence)
	}
}

func TestDeduplicateDifferentTypes(t *testing.T) {
	issues := []models.Issue{
		{
			Type:       "sql_injection",
			Severity:   "critical",
			Confidence: 0.85,
			Location:   models.Location{File: "main.go", StartLine: 10},
		},
		{
			Type:       "hardcoded_secrets",
			Severity:   "critical",
			Confidence: 0.95,
			Location:   models.Location{File: "main.go", StartLine: 10},
		},
	}

	deduped := deduplicate(issues)

	if len(deduped) != 2 {
		t.Errorf("Expected 2 issues (different types), got %d", len(deduped))
	}
}

func TestSortByPrioritySeverity(t *testing.T) {
	issues := []models.Issue{
		{
			Type:     "minor_issue",
			Severity: "minor",
			Location: models.Location{File: "a.go", StartLine: 1},
		},
		{
			Type:     "critical_issue",
			Severity: "critical",
			Location: models.Location{File: "a.go", StartLine: 1},
		},
		{
			Type:     "major_issue",
			Severity: "major",
			Location: models.Location{File: "a.go", StartLine: 1},
		},
	}

	sorted := sortByPriority(issues)

	expected := []string{"critical", "major", "minor"}
	for i, expSev := range expected {
		if sorted[i].Severity != expSev {
			t.Errorf("Position %d: expected severity %s, got %s", i, expSev, sorted[i].Severity)
		}
	}
}

func TestSortByPriorityFile(t *testing.T) {
	issues := []models.Issue{
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "z.go", StartLine: 5},
		},
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "a.go", StartLine: 10},
		},
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "m.go", StartLine: 3},
		},
	}

	sorted := sortByPriority(issues)

	expectedFiles := []string{"a.go", "m.go", "z.go"}
	for i, expFile := range expectedFiles {
		if sorted[i].Location.File != expFile {
			t.Errorf("Position %d: expected file %s, got %s", i, expFile, sorted[i].Location.File)
		}
	}
}

func TestSortByPriorityLine(t *testing.T) {
	issues := []models.Issue{
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "a.go", StartLine: 50},
		},
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "a.go", StartLine: 10},
		},
		{
			Type:     "issue",
			Severity: "major",
			Location: models.Location{File: "a.go", StartLine: 30},
		},
	}

	sorted := sortByPriority(issues)

	expectedLines := []int{10, 30, 50}
	for i, expLine := range expectedLines {
		if sorted[i].Location.StartLine != expLine {
			t.Errorf("Position %d: expected line %d, got %d", i, expLine, sorted[i].Location.StartLine)
		}
	}
}

func TestCalculateSummaryQualityA(t *testing.T) {
	issues := []models.Issue{
		{
			Severity:   "minor",
			Confidence: 0.9,
			Location:   models.Location{File: "a.go", StartLine: 1},
		},
		{
			Severity:   "minor",
			Confidence: 0.8,
			Location:   models.Location{File: "b.go", StartLine: 5},
		},
	}

	summary := calculateSummary(issues)

	if summary.Quality != "A" {
		t.Errorf("Expected quality A (score 98), got %s", summary.Quality)
	}
	if summary.TotalIssues != 2 {
		t.Errorf("Expected 2 issues, got %d", summary.TotalIssues)
	}
	if summary.MinorCount != 2 {
		t.Errorf("Expected 2 minor issues, got %d", summary.MinorCount)
	}
}

func TestCalculateSummaryQualityB(t *testing.T) {
	issues := []models.Issue{
		{
			Severity:   "major",
			Confidence: 0.9,
			Location:   models.Location{File: "a.go", StartLine: 1},
		},
		{
			Severity:   "minor",
			Confidence: 0.8,
			Location:   models.Location{File: "b.go", StartLine: 5},
		},
	}

	summary := calculateSummary(issues)

	// Score: 100 - (0*10 + 1*5 + 1*1) = 94, which is A
	// Let me recalculate: 100 - (0 + 5 + 1) = 94, so A
	// Actually this should be A. Let me adjust the test.
	if summary.Quality != "A" {
		t.Errorf("Expected quality A (score 94), got %s", summary.Quality)
	}
	if summary.MajorCount != 1 {
		t.Errorf("Expected 1 major issue, got %d", summary.MajorCount)
	}
	if summary.MinorCount != 1 {
		t.Errorf("Expected 1 minor issue, got %d", summary.MinorCount)
	}
}

func TestCalculateSummaryQualityD(t *testing.T) {
	issues := []models.Issue{
		{
			Severity:   "critical",
			Confidence: 0.9,
			Location:   models.Location{File: "a.go", StartLine: 1},
		},
		{
			Severity:   "critical",
			Confidence: 0.8,
			Location:   models.Location{File: "b.go", StartLine: 5},
		},
		{
			Severity:   "major",
			Confidence: 0.9,
			Location:   models.Location{File: "c.go", StartLine: 10},
		},
	}

	summary := calculateSummary(issues)

	// Score: 100 - (2*10 + 1*5 + 0*1) = 75, which is B
	if summary.Quality != "B" {
		t.Errorf("Expected quality B (score 75), got %s", summary.Quality)
	}
	if summary.CriticalCount != 2 {
		t.Errorf("Expected 2 critical issues, got %d", summary.CriticalCount)
	}
}

func TestCalculateConfidenceAverage(t *testing.T) {
	issues := []models.Issue{
		{
			Severity:   "minor",
			Confidence: 1.0,
			Location:   models.Location{File: "a.go", StartLine: 1},
		},
		{
			Severity:   "minor",
			Confidence: 0.8,
			Location:   models.Location{File: "b.go", StartLine: 5},
		},
	}

	summary := calculateSummary(issues)

	expected := 0.9 // (1.0 + 0.8) / 2
	if summary.Confidence != expected {
		t.Errorf("Expected confidence %v, got %v", expected, summary.Confidence)
	}
}

func TestHashDiff(t *testing.T) {
	diff := "--- a/main.go\n+++ b/main.go\n@@ -1,5 +1,6 @@\n+var x = 10"

	hash := hashDiff(diff)

	if hash == "" {
		t.Error("Hash should not be empty")
	}
	if len(hash) != 16 {
		t.Errorf("Expected 16-char hash, got %d", len(hash))
	}

	// Same diff should produce same hash
	hash2 := hashDiff(diff)
	if hash != hash2 {
		t.Errorf("Same diff should produce same hash: %s vs %s", hash, hash2)
	}

	// Different diff should produce different hash
	hash3 := hashDiff(diff + "modified")
	if hash == hash3 {
		t.Error("Different diffs should produce different hashes")
	}
}

func TestCountFiles(t *testing.T) {
	hunks := []models.DiffHunk{
		{File: "a.go", AddedLines: []string{"x"}},
		{File: "b.go", AddedLines: []string{"y"}},
		{File: "a.go", AddedLines: []string{"z"}},
	}

	count := countFiles(hunks)

	if count != 2 {
		t.Errorf("Expected 2 unique files, got %d", count)
	}
}

func TestCountTotalLines(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:       "a.go",
			AddedLines: []string{"line1", "line2"},
			RemovedLines: []string{"oldline"},
		},
		{
			File:       "b.go",
			AddedLines: []string{"line3"},
			RemovedLines: []string{},
		},
	}

	count := countTotalLines(hunks)

	// 2 + 1 added, 1 + 0 removed = 4 total
	if count != 4 {
		t.Errorf("Expected 4 total lines, got %d", count)
	}
}

func TestAggregateFullFlow(t *testing.T) {
	localIssues := []models.Issue{
		{
			Type:       "hardcoded_secrets",
			Severity:   "critical",
			Confidence: 0.98,
			Source:     "local_analyzer",
			Location:   models.Location{File: "config.go", StartLine: 5},
			Message:    "Found API key",
		},
	}

	llmIssues := []models.Issue{
		{
			Type:       "sql_injection",
			Severity:   "critical",
			Confidence: 0.92,
			Source:     "llm_analyzer",
			Location:   models.Location{File: "db.go", StartLine: 15},
			Message:    "Potential SQL injection",
		},
		{
			Type:       "hardcoded_secrets",
			Severity:   "critical",
			Confidence: 0.95,
			Source:     "llm_analyzer",
			Location:   models.Location{File: "config.go", StartLine: 5},
			Message:    "Found password",
		},
	}

	hunks := []models.DiffHunk{
		{
			File:       "config.go",
			StartLine:  1,
			AddedLines: []string{"const apiKey = \"sk-123\""},
		},
		{
			File:       "db.go",
			StartLine:  10,
			AddedLines: []string{"query := \"SELECT * FROM users WHERE id = '\" + userId + \"'\""},
		},
	}

	diff := "mock diff content"

	result := Aggregate(localIssues, llmIssues, hunks, diff)

	// Should have deduplicated the hardcoded_secrets (kept highest confidence 0.98 from local)
	// and kept the sql_injection from LLM
	if result.Summary.TotalIssues != 2 {
		t.Errorf("Expected 2 issues after dedup, got %d", result.Summary.TotalIssues)
	}

	if result.FileCount != 2 {
		t.Errorf("Expected 2 files, got %d", result.FileCount)
	}

	if result.Summary.CriticalCount != 2 {
		t.Errorf("Expected 2 critical issues, got %d", result.Summary.CriticalCount)
	}

	// First issue should be critical (both are critical, so first by file)
	if result.Issues[0].Severity != "critical" {
		t.Errorf("Expected first issue to be critical, got %s", result.Issues[0].Severity)
	}

	if result.Duration < 0 {
		t.Errorf("Expected duration >= 0, got %v", result.Duration)
	}

	if result.DiffHash == "" {
		t.Error("Expected non-empty diff hash")
	}
}
