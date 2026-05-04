package formatter

import (
	"code-review-agent/internal/models"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestFormatJSON(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "abc123",
		FileCount:  2,
		TotalLines: 50,
		Issues: []models.Issue{
			{
				Type:       "hardcoded_secrets",
				Severity:   "critical",
				Confidence: 0.95,
				Location:   models.Location{File: "config.go", StartLine: 10},
				Message:    "Found API key",
				Source:     "local_analyzer",
			},
		},
		Summary: models.Summary{
			CriticalCount: 1,
			MajorCount:    0,
			MinorCount:    0,
			TotalIssues:   1,
			Quality:       "B",
			Confidence:    0.95,
		},
		Duration: 25.5,
	}

	output := FormatJSON(result)
	if output == "" {
		t.Error("FormatJSON returned empty string")
	}

	var parsed models.AnalysisResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("FormatJSON output is not valid JSON: %v", err)
	}

	if parsed.FileCount != 2 {
		t.Errorf("Expected 2 files, got %d", parsed.FileCount)
	}
	if parsed.Summary.TotalIssues != 1 {
		t.Errorf("Expected 1 issue, got %d", parsed.Summary.TotalIssues)
	}
}

func TestFormatCLI(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp: time.Now(),
		DiffHash:  "abc123",
		FileCount: 1,
		Issues: []models.Issue{
			{
				Type:       "sql_injection",
				Severity:   "critical",
				Confidence: 0.90,
				Location:   models.Location{File: "db.go", StartLine: 42},
				Message:    "SQL injection detected",
				Source:     "llm_analyzer",
			},
		},
		Summary: models.Summary{
			CriticalCount: 1,
			TotalIssues:   1,
			Quality:       "D",
			Confidence:    0.90,
		},
		Duration: 150.0,
	}

	output := FormatCLI(result)
	if output == "" {
		t.Error("FormatCLI returned empty string")
	}

	if !strings.Contains(output, "Code Review Analysis") {
		t.Error("FormatCLI missing header")
	}
	if !strings.Contains(output, "sql_injection") {
		t.Error("FormatCLI missing issue type")
	}
	if !strings.Contains(output, "critical") {
		t.Error("FormatCLI missing severity")
	}
}

func TestFormatMarkdown(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp:  time.Now(),
		FileCount:  3,
		TotalLines: 100,
		Issues: []models.Issue{
			{
				Type:       "todo_comment",
				Severity:   "minor",
				Confidence: 0.99,
				Location:   models.Location{File: "main.go", StartLine: 5},
				Message:    "TODO comment found",
				Suggestion: "Remove or implement TODO",
				Source:     "local_analyzer",
			},
		},
		Summary: models.Summary{
			MinorCount:  1,
			TotalIssues: 1,
			Quality:     "A",
			Confidence:  0.99,
		},
		Duration: 10.0,
	}

	output := FormatMarkdown(result)
	if output == "" {
		t.Error("FormatMarkdown returned empty string")
	}

	if !strings.Contains(output, "# Code Review Report") {
		t.Error("FormatMarkdown missing header")
	}
	if !strings.Contains(output, "## Summary") {
		t.Error("FormatMarkdown missing summary section")
	}
	if !strings.Contains(output, "todo_comment") {
		t.Error("FormatMarkdown missing issue type")
	}
}
