package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"

	"code-review-agent/internal/models"
)

// Store wraps a SQLite database connection.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) a SQLite database at the given path,
// enables WAL mode and busy timeout, then runs Migrate().
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	store := &Store{db: db}
	if err := store.Migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// Migrate creates the required tables if they do not exist (idempotent).
func (s *Store) Migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS repositories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL,
    path       TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS analyses (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id        INTEGER NOT NULL REFERENCES repositories(id),
    diff_hash      TEXT NOT NULL UNIQUE,
    timestamp      DATETIME NOT NULL,
    file_count     INTEGER NOT NULL,
    total_lines    INTEGER NOT NULL,
    duration_ms    REAL NOT NULL,
    critical_count INTEGER NOT NULL DEFAULT 0,
    major_count    INTEGER NOT NULL DEFAULT 0,
    minor_count    INTEGER NOT NULL DEFAULT 0,
    total_issues   INTEGER NOT NULL DEFAULT 0,
    quality        TEXT NOT NULL DEFAULT '',
    avg_confidence REAL NOT NULL DEFAULT 0.0,
    created_at     DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS issues (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    analysis_id INTEGER NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    issue_id    TEXT NOT NULL,
    type        TEXT NOT NULL,
    severity    TEXT NOT NULL,
    file        TEXT NOT NULL,
    start_line  INTEGER NOT NULL,
    end_line    INTEGER NOT NULL,
    message     TEXT NOT NULL,
    suggestion  TEXT NOT NULL,
    confidence  REAL NOT NULL,
    source      TEXT NOT NULL
);
`

	_, err := s.db.Exec(schema)
	return err
}

// UpsertRepository inserts or updates a repository, returning its ID.
func (s *Store) UpsertRepository(name, path string) (int64, error) {
	query := `
INSERT INTO repositories (name, path, updated_at)
VALUES (?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(path) DO UPDATE SET name=excluded.name, updated_at=excluded.updated_at
`

	res, err := s.db.Exec(query, name, path)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// CreateAnalysis inserts an analysis and its associated issues,
// returning the analysis ID.
func (s *Store) CreateAnalysis(repoID int64, result *models.AnalysisResult) (int64, error) {
	analysisQuery := `
INSERT INTO analyses (repo_id, diff_hash, timestamp, file_count, total_lines, duration_ms,
                      critical_count, major_count, minor_count, total_issues, quality, avg_confidence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	res, err := s.db.Exec(
		analysisQuery,
		repoID,
		result.DiffHash,
		result.Timestamp,
		result.FileCount,
		result.TotalLines,
		result.Duration,
		result.Summary.CriticalCount,
		result.Summary.MajorCount,
		result.Summary.MinorCount,
		result.Summary.TotalIssues,
		result.Summary.Quality,
		result.Summary.Confidence,
	)

	if err != nil {
		return 0, fmt.Errorf("failed to insert analysis: %w", err)
	}

	analysisID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	issueQuery := `
INSERT INTO issues (analysis_id, issue_id, type, severity, file, start_line, end_line,
                    message, suggestion, confidence, source)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	for _, issue := range result.Issues {
		_, err := s.db.Exec(
			issueQuery,
			analysisID,
			issue.ID,
			issue.Type,
			issue.Severity,
			issue.Location.File,
			issue.Location.StartLine,
			issue.Location.EndLine,
			issue.Message,
			issue.Suggestion,
			issue.Confidence,
			issue.Source,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to insert issue: %w", err)
		}
	}

	return analysisID, nil
}

// UpdateAnalysisResult updates the summary and duration of an existing analysis
// identified by diff_hash. Does not affect already-inserted issues.
func (s *Store) UpdateAnalysisResult(diffHash string, result *models.AnalysisResult) error {
	query := `
UPDATE analyses
SET timestamp = ?, file_count = ?, total_lines = ?, duration_ms = ?,
    critical_count = ?, major_count = ?, minor_count = ?, total_issues = ?,
    quality = ?, avg_confidence = ?
WHERE diff_hash = ?
`

	_, err := s.db.Exec(
		query,
		result.Timestamp,
		result.FileCount,
		result.TotalLines,
		result.Duration,
		result.Summary.CriticalCount,
		result.Summary.MajorCount,
		result.Summary.MinorCount,
		result.Summary.TotalIssues,
		result.Summary.Quality,
		result.Summary.Confidence,
		diffHash,
	)

	return err
}

// GetAnalysisByHash returns a single analysis by its diff hash, or nil if not found.
func (s *Store) GetAnalysisByHash(hash string) (*models.AnalysisResult, error) {
	query := `
SELECT diff_hash, timestamp, file_count, total_lines, duration_ms,
       critical_count, major_count, minor_count, total_issues, quality, avg_confidence
FROM analyses
WHERE diff_hash = ?
LIMIT 1
`
	row := s.db.QueryRow(query, hash)
	var result models.AnalysisResult
	err := row.Scan(
		&result.DiffHash,
		&result.Timestamp,
		&result.FileCount,
		&result.TotalLines,
		&result.Duration,
		&result.Summary.CriticalCount,
		&result.Summary.MajorCount,
		&result.Summary.MinorCount,
		&result.Summary.TotalIssues,
		&result.Summary.Quality,
		&result.Summary.Confidence,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query analysis by hash: %w", err)
	}
	result.Issues = []models.Issue{}
	return &result, nil
}

// ListRecentAnalyses returns the most recent analyses across all repos, newest first.
func (s *Store) ListRecentAnalyses(limit int) ([]models.AnalysisResult, error) {
	query := `
SELECT diff_hash, timestamp, file_count, total_lines, duration_ms,
       critical_count, major_count, minor_count, total_issues, quality, avg_confidence
FROM analyses
ORDER BY timestamp DESC
LIMIT ?
`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent analyses: %w", err)
	}
	defer rows.Close()

	var results []models.AnalysisResult
	for rows.Next() {
		var result models.AnalysisResult
		err := rows.Scan(
			&result.DiffHash,
			&result.Timestamp,
			&result.FileCount,
			&result.TotalLines,
			&result.Duration,
			&result.Summary.CriticalCount,
			&result.Summary.MajorCount,
			&result.Summary.MinorCount,
			&result.Summary.TotalIssues,
			&result.Summary.Quality,
			&result.Summary.Confidence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analysis row: %w", err)
		}
		result.Issues = []models.Issue{}
		results = append(results, result)
	}
	return results, rows.Err()
}

// ListAnalysesForRepo returns all analyses for a repository (without nested issues).
func (s *Store) ListAnalysesForRepo(repoID int64) ([]models.AnalysisResult, error) {
	query := `
SELECT id, diff_hash, timestamp, file_count, total_lines, duration_ms,
       critical_count, major_count, minor_count, total_issues, quality, avg_confidence
FROM analyses
WHERE repo_id = ?
ORDER BY timestamp DESC
`

	rows, err := s.db.Query(query, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query analyses: %w", err)
	}
	defer rows.Close()

	var results []models.AnalysisResult
	for rows.Next() {
		var result models.AnalysisResult
		var id int64
		err := rows.Scan(
			&id, // id (not used but must scan)
			&result.DiffHash,
			&result.Timestamp,
			&result.FileCount,
			&result.TotalLines,
			&result.Duration,
			&result.Summary.CriticalCount,
			&result.Summary.MajorCount,
			&result.Summary.MinorCount,
			&result.Summary.TotalIssues,
			&result.Summary.Quality,
			&result.Summary.Confidence,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analysis row: %w", err)
		}
		result.Issues = []models.Issue{}
		results = append(results, result)
	}

	return results, rows.Err()
}
