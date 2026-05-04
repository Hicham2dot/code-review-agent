package cache

import (
	"code-review-agent/internal/models"
	"os"
	"testing"
	"time"
)

func TestSetAndGet(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp:  time.Now(),
		DiffHash:   "test_hash_123",
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
			TotalIssues:   1,
			Quality:       "B",
			Confidence:    0.95,
		},
		Duration: 25.5,
	}

	hash := "test_123"
	ttl := 3600

	if err := Set(hash, result, ttl); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved := Get(hash, ttl)
	if retrieved == nil {
		t.Error("Get returned nil")
	}

	if retrieved.FileCount != result.FileCount {
		t.Errorf("Expected FileCount %d, got %d", result.FileCount, retrieved.FileCount)
	}

	if len(retrieved.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(retrieved.Issues))
	}

	if retrieved.Summary.TotalIssues != 1 {
		t.Errorf("Expected 1 total issue, got %d", retrieved.Summary.TotalIssues)
	}
}

func TestGetNonExistent(t *testing.T) {
	retrieved := Get("nonexistent_hash_12345", 3600)
	if retrieved != nil {
		t.Error("Get should return nil for non-existent cache entry")
	}
}

func TestTTLExpiration(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp: time.Now(),
		DiffHash:  "ttl_test",
		FileCount: 1,
		Summary: models.Summary{
			TotalIssues: 0,
			Quality:     "A",
			Confidence:  1.0,
		},
	}

	hash := "ttl_expired"
	ttl := 1 // 1 second

	if err := Set(hash, result, ttl); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	time.Sleep(2 * time.Second)

	retrieved := Get(hash, ttl)
	if retrieved != nil {
		t.Error("Get should return nil for expired cache entry")
	}
}

func TestClear(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp: time.Now(),
		FileCount: 1,
		Summary: models.Summary{
			Quality: "A",
		},
	}

	if err := Set("cache_clear_test", result, 3600); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved := Get("cache_clear_test", 3600)
	if retrieved == nil {
		t.Error("Entry should exist before Clear")
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	cacheDir := cacheDirectory()
	if _, err := os.Stat(cacheDir); err == nil {
		t.Errorf("Cache directory should be deleted after Clear")
	}
}

func TestCacheDirectoryCreation(t *testing.T) {
	result := &models.AnalysisResult{
		Timestamp: time.Now(),
		FileCount: 1,
		Summary: models.Summary{
			Quality: "A",
		},
	}

	cacheDir := cacheDirectory()
	os.RemoveAll(cacheDir)

	if err := Set("dir_create_test", result, 3600); err != nil {
		t.Fatalf("Set should create cache directory: %v", err)
	}

	if _, err := os.Stat(cacheDir); err != nil {
		t.Errorf("Cache directory should exist after Set: %v", err)
	}

	os.RemoveAll(cacheDir)
}
