package models

import "time"

type Issue struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Severity   string   `json:"severity"`
	Location   Location `json:"location"`
	Message    string   `json:"message"`
	Suggestion string   `json:"suggestion"`
	Confidence float64  `json:"confidence"`
	Source     string   `json:"source"`
}

type Location struct {
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

type AnalysisResult struct {
	Timestamp  time.Time `json:"timestamp"`
	DiffHash   string    `json:"diff_hash"`
	FileCount  int       `json:"file_count"`
	TotalLines int       `json:"total_lines"`
	Issues     []Issue   `json:"issues"`
	Summary    Summary   `json:"summary"`
	Duration   float64   `json:"duration_ms"`
}

type Summary struct {
	CriticalCount int     `json:"critical_count"`
	MajorCount    int     `json:"major_count"`
	MinorCount    int     `json:"minor_count"`
	TotalIssues   int     `json:"total_issues"`
	Quality       string  `json:"quality"`
	Confidence    float64 `json:"avg_confidence"`
}

type DiffHunk struct {
	File         string
	StartLine    int
	EndLine      int
	RemovedLines []string
	AddedLines   []string
	Context      string
}
