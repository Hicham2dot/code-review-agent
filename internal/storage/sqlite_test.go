package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"code-review-agent/internal/models"
)

func TestNewStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatal("Database file not created")
	}
}

func TestMigrate(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	row := store.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='repositories'")
	var name string
	err = row.Scan(&name)
	if err != nil {
		t.Fatal("repositories table not created")
	}

	row = store.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='analyses'")
	err = row.Scan(&name)
	if err != nil {
		t.Fatal("analyses table not created")
	}

	row = store.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='issues'")
	err = row.Scan(&name)
	if err != nil {
		t.Fatal("issues table not created")
	}
}

func TestUpsertRepository(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	id1, err := store.UpsertRepository("test-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	if id1 <= 0 {
		t.Fatalf("expected positive id, got %d", id1)
	}

	id2, err := store.UpsertRepository("updated-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("UpsertRepository (update) failed: %v", err)
	}

	if id1 != id2 {
		t.Fatalf("expected same id for upsert, got %d != %d", id1, id2)
	}

	var repoName string
	row := store.db.QueryRow("SELECT name FROM repositories WHERE id = ?", id1)
	err = row.Scan(&repoName)
	if err != nil {
		t.Fatal("failed to query repository")
	}

	if repoName != "updated-repo" {
		t.Fatalf("expected name 'updated-repo', got '%s'", repoName)
	}
}

func TestCreateAnalysis(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	repoID, err := store.UpsertRepository("test-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	result := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "abc123def456",
		FileCount:  2,
		TotalLines: 50,
		Duration:   123.45,
		Issues: []models.Issue{
			{
				ID:         "sec-1",
				Type:       "hardcoded_secrets",
				Severity:   "critical",
				Location:   models.Location{File: "app.go", StartLine: 10, EndLine: 10},
				Message:    "hardcoded API key detected",
				Suggestion: "use environment variable",
				Confidence: 0.98,
				Source:     "local_analyzer",
			},
			{
				ID:         "sql-1",
				Type:       "sql_injection",
				Severity:   "major",
				Location:   models.Location{File: "db.go", StartLine: 20, EndLine: 20},
				Message:    "SQL concatenation detected",
				Suggestion: "use parameterized queries",
				Confidence: 0.85,
				Source:     "local_analyzer",
			},
		},
		Summary: models.Summary{
			CriticalCount: 1,
			MajorCount:    1,
			MinorCount:    0,
			TotalIssues:   2,
			Quality:       "low",
			Confidence:    0.915,
		},
	}

	analysisID, err := store.CreateAnalysis(repoID, result)
	if err != nil {
		t.Fatalf("CreateAnalysis failed: %v", err)
	}

	if analysisID <= 0 {
		t.Fatalf("expected positive analysis id, got %d", analysisID)
	}

	var issueCount int
	row := store.db.QueryRow("SELECT COUNT(*) FROM issues WHERE analysis_id = ?", analysisID)
	err = row.Scan(&issueCount)
	if err != nil {
		t.Fatal("failed to count issues")
	}

	if issueCount != 2 {
		t.Fatalf("expected 2 issues, got %d", issueCount)
	}
}

func TestUpdateAnalysisResult(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	repoID, err := store.UpsertRepository("test-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	result := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "xyz789",
		FileCount:  1,
		TotalLines: 10,
		Duration:   50.0,
		Issues:     []models.Issue{},
		Summary: models.Summary{
			CriticalCount: 0,
			MajorCount:    0,
			MinorCount:    0,
			TotalIssues:   0,
			Quality:       "good",
			Confidence:    1.0,
		},
	}

	_, err = store.CreateAnalysis(repoID, result)
	if err != nil {
		t.Fatalf("CreateAnalysis failed: %v", err)
	}

	updatedResult := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "xyz789",
		FileCount:  2,
		TotalLines: 20,
		Duration:   75.0,
		Issues:     []models.Issue{},
		Summary: models.Summary{
			CriticalCount: 0,
			MajorCount:    0,
			MinorCount:    1,
			TotalIssues:   1,
			Quality:       "fair",
			Confidence:    0.9,
		},
	}

	err = store.UpdateAnalysisResult("xyz789", updatedResult)
	if err != nil {
		t.Fatalf("UpdateAnalysisResult failed: %v", err)
	}

	var duration float64
	var minorCount int
	row := store.db.QueryRow("SELECT duration_ms, minor_count FROM analyses WHERE diff_hash = ?", "xyz789")
	err = row.Scan(&duration, &minorCount)
	if err != nil {
		t.Fatal("failed to query updated analysis")
	}

	if duration != 75.0 {
		t.Fatalf("expected duration 75.0, got %f", duration)
	}

	if minorCount != 1 {
		t.Fatalf("expected minor_count 1, got %d", minorCount)
	}
}

func TestListAnalysesForRepo(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	defer store.Close()

	repoID, err := store.UpsertRepository("test-repo", "/path/to/repo")
	if err != nil {
		t.Fatalf("UpsertRepository failed: %v", err)
	}

	result1 := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "hash1",
		FileCount:  1,
		TotalLines: 10,
		Duration:   50.0,
		Issues:     []models.Issue{},
		Summary:    models.Summary{TotalIssues: 0, Quality: "good"},
	}

	result2 := &models.AnalysisResult{
		Timestamp:  time.Now().Add(1 * time.Second),
		DiffHash:   "hash2",
		FileCount:  2,
		TotalLines: 20,
		Duration:   75.0,
		Issues:     []models.Issue{},
		Summary:    models.Summary{TotalIssues: 1, Quality: "fair"},
	}

	_, err = store.CreateAnalysis(repoID, result1)
	if err != nil {
		t.Fatalf("CreateAnalysis failed: %v", err)
	}

	_, err = store.CreateAnalysis(repoID, result2)
	if err != nil {
		t.Fatalf("CreateAnalysis failed: %v", err)
	}

	results, err := store.ListAnalysesForRepo(repoID)
	if err != nil {
		t.Fatalf("ListAnalysesForRepo failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 analyses, got %d", len(results))
	}

	if results[0].DiffHash != "hash2" || results[1].DiffHash != "hash1" {
		t.Fatal("expected results sorted by timestamp DESC (hash2, hash1)")
	}

	if results[0].FileCount != 2 {
		t.Fatalf("expected first result file_count 2, got %d", results[0].FileCount)
	}

	if results[1].FileCount != 1 {
		t.Fatalf("expected second result file_count 1, got %d", results[1].FileCount)
	}
}
