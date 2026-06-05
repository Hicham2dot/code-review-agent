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
CREATE TABLE IF NOT EXISTS users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'user',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT NOT NULL UNIQUE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

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
    user_id        INTEGER REFERENCES users(id),
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
// GetIssuesByAnalysisHash returns all issues associated with an analysis diff hash.
func (s *Store) GetIssuesByAnalysisHash(hash string) ([]models.Issue, error) {
	query := `
SELECT i.issue_id, i.type, i.severity, i.file, i.start_line,
       i.end_line, i.message, i.suggestion, i.confidence, i.source
FROM issues i
JOIN analyses a ON i.analysis_id = a.id
WHERE a.diff_hash = ?
ORDER BY i.severity, i.confidence DESC
`
	rows, err := s.db.Query(query, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to query issues: %w", err)
	}
	defer rows.Close()

	var issues []models.Issue
	for rows.Next() {
		var iss models.Issue
		err := rows.Scan(
			&iss.ID,
			&iss.Type,
			&iss.Severity,
			&iss.Location.File,
			&iss.Location.StartLine,
			&iss.Location.EndLine,
			&iss.Message,
			&iss.Suggestion,
			&iss.Confidence,
			&iss.Source,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan issue row: %w", err)
		}
		issues = append(issues, iss)
	}
	if issues == nil {
		issues = []models.Issue{}
	}
	return issues, rows.Err()
}

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
	issues, err := s.GetIssuesByAnalysisHash(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to load issues for analysis: %w", err)
	}
	result.Issues = issues
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

// UserInfo holds public user data.
type UserInfo struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
	Analyses  int    `json:"analyses_count"`
}

// CreateUser inserts a new user with a bcrypt-hashed password and role.
func (s *Store) CreateUser(username, passwordHash, role string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateUserPassword updates the password hash for the given username.
func (s *Store) UpdateUserPassword(username, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE users SET password_hash = ? WHERE username = ?`, passwordHash, username)
	return err
}

// GetUserByUsername returns id, passwordHash and role for the given username.
func (s *Store) GetUserByUsername(username string) (id int64, passwordHash, role string, err error) {
	row := s.db.QueryRow(`SELECT id, password_hash, role FROM users WHERE username = ?`, username)
	err = row.Scan(&id, &passwordHash, &role)
	if err == sql.ErrNoRows {
		return 0, "", "", nil
	}
	return
}

// GetUserRole returns the role of a user by ID.
func (s *Store) GetUserRole(userID int64) (string, error) {
	var role string
	err := s.db.QueryRow(`SELECT role FROM users WHERE id = ?`, userID).Scan(&role)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return role, err
}

// UserCount returns the number of users in the database.
func (s *Store) UserCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}

// ListUsers returns all users with their analyses count.
func (s *Store) ListUsers() ([]UserInfo, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.username, u.role, u.created_at,
		       COUNT(a.id) AS analyses_count
		FROM users u
		LEFT JOIN analyses a ON a.user_id = u.id
		GROUP BY u.id
		ORDER BY u.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UserInfo
	for rows.Next() {
		var u UserInfo
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt, &u.Analyses); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	if users == nil {
		users = []UserInfo{}
	}
	return users, rows.Err()
}

// ListAnalysesForUser returns recent analyses submitted by a specific user.
func (s *Store) ListAnalysesForUser(userID int64, limit int) ([]models.AnalysisResult, error) {
	rows, err := s.db.Query(`
		SELECT diff_hash, timestamp, file_count, total_lines, duration_ms,
		       critical_count, major_count, minor_count, total_issues, quality, avg_confidence
		FROM analyses
		WHERE user_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.AnalysisResult
	for rows.Next() {
		var r models.AnalysisResult
		if err := rows.Scan(&r.DiffHash, &r.Timestamp, &r.FileCount, &r.TotalLines,
			&r.Duration, &r.Summary.CriticalCount, &r.Summary.MajorCount,
			&r.Summary.MinorCount, &r.Summary.TotalIssues, &r.Summary.Quality, &r.Summary.Confidence,
		); err != nil {
			return nil, err
		}
		r.Issues = []models.Issue{}
		results = append(results, r)
	}
	if results == nil {
		results = []models.AnalysisResult{}
	}
	return results, rows.Err()
}

// CreateSession inserts a session token for a user.
func (s *Store) CreateSession(token string, userID int64, expiresAt string) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt,
	)
	return err
}

// GetSessionUserID returns the user ID for a valid (non-expired) session token.
// Returns 0 if not found or expired.
func (s *Store) GetSessionUserID(token string) (int64, error) {
	var userID int64
	err := s.db.QueryRow(
		`SELECT user_id FROM sessions WHERE token = ? AND expires_at > CURRENT_TIMESTAMP`,
		token,
	).Scan(&userID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return userID, err
}

// DeleteSession removes a session by token.
func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
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
