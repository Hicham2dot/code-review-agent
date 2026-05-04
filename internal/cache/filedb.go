package cache

import (
	"code-review-agent/internal/models"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type CacheEntry struct {
	Result    *models.AnalysisResult `json:"result"`
	Timestamp time.Time              `json:"timestamp"`
	TTL       int                    `json:"ttl_seconds"`
}

func Get(hash string, ttlSeconds int) *models.AnalysisResult {
	cacheDir := cacheDirectory()
	filePath := filepath.Join(cacheDir, hash+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil
	}

	// Check TTL
	if ttlSeconds > 0 {
		age := time.Since(entry.Timestamp).Seconds()
		if age > float64(ttlSeconds) {
			os.Remove(filePath)
			return nil
		}
	}

	return entry.Result
}

func Set(hash string, result *models.AnalysisResult, ttlSeconds int) error {
	cacheDir := cacheDirectory()
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	entry := CacheEntry{
		Result:    result,
		Timestamp: time.Now(),
		TTL:       ttlSeconds,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	filePath := filepath.Join(cacheDir, hash+".json")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func Clear() error {
	cacheDir := cacheDirectory()
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

func cacheDirectory() string {
	if h := os.Getenv("HOME"); h != "" {
		return filepath.Join(h, ".cache", "code-review-agent")
	}
	return filepath.Join(os.TempDir(), "code-review-agent-cache")
}
