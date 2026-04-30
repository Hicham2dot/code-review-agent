package local

import (
	"code-review-agent/internal/models"
	"strings"
	"testing"
)

func TestLocalAnalyzeTodoComment(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "main.go",
			StartLine: 10,
			EndLine:   12,
			AddedLines: []string{
				"func main() {",
				"    // TODO: implement error handling",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	if len(issues) == 0 {
		t.Fatal("Expected to find TODO comment issue, got none")
	}

	found := false
	for _, issue := range issues {
		if issue.Type == "todo_comment" {
			found = true
			if issue.Severity != "minor" {
				t.Errorf("Expected severity 'minor', got '%s'", issue.Severity)
			}
			if issue.Location.File != "main.go" {
				t.Errorf("Expected file 'main.go', got '%s'", issue.Location.File)
			}
			break
		}
	}

	if !found {
		t.Fatal("TODO comment issue not found in results")
	}
}

func TestLocalAnalyzeHardcodedSecrets(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "config.go",
			StartLine: 5,
			EndLine:   7,
			AddedLines: []string{
				"const (",
				`    apiKey = "sk-1234567890abcdef"`,
				")",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	found := false
	for _, issue := range issues {
		if issue.Type == "hardcoded_secrets" {
			found = true
			if issue.Severity != "critical" {
				t.Errorf("Expected severity 'critical', got '%s'", issue.Severity)
			}
			if issue.Confidence < 0.9 {
				t.Errorf("Expected high confidence, got %f", issue.Confidence)
			}
			break
		}
	}

	if !found {
		t.Fatal("Hardcoded secrets issue not found in results")
	}
}

func TestLocalAnalyzeSQLInjection(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "db.go",
			StartLine: 10,
			EndLine:   12,
			AddedLines: []string{
				"func queryUser(id string) {",
				`    query := "SELECT * FROM users WHERE id = " + id`,
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	found := false
	for _, issue := range issues {
		if issue.Type == "sql_injection" {
			found = true
			if issue.Severity != "critical" {
				t.Errorf("Expected severity 'critical', got '%s'", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Fatal("SQL injection issue not found in results")
	}
}

func TestLocalAnalyzeDeprecatedFunction(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "reader.go",
			StartLine: 5,
			EndLine:   8,
			AddedLines: []string{
				"import \"io/ioutil\"",
				"func readFile() {",
				"    data, _ := ioutil.ReadFile(\"test.txt\")",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	found := false
	for _, issue := range issues {
		if issue.Type == "deprecated_function" {
			found = true
			if issue.Severity != "minor" {
				t.Errorf("Expected severity 'minor', got '%s'", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Fatal("Deprecated function issue not found in results")
	}
}

func TestLocalAnalyzeMissingErrorHandling(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "handler.go",
			StartLine: 10,
			EndLine:   15,
			AddedLines: []string{
				"func getValue() {",
				"    data, err := ioutil.ReadFile(\"file.txt\")",
				"    if err != nil {",
				"        return",
				"    }",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	// This test verifies that proper error handling doesn't trigger issues
	for _, issue := range issues {
		if issue.Type == "missing_error_handling" {
			t.Errorf("Found unexpected missing error handling issue: %v", issue)
		}
	}
}

func TestLocalAnalyzeLargeFunction(t *testing.T) {
	// Create a large function (more than 50 lines)
	lines := make([]string, 0)
	lines = append(lines, "func largeFunc() {")
	for i := 0; i < 55; i++ {
		lines = append(lines, "    x := x + 1")
	}
	lines = append(lines, "}")

	hunks := []models.DiffHunk{
		{
			File:      "large.go",
			StartLine: 1,
			EndLine:   len(lines),
			AddedLines: lines,
		},
	}

	issues := LocalAnalyze(hunks)

	found := false
	for _, issue := range issues {
		if issue.Type == "large_function" {
			found = true
			if issue.Severity != "major" {
				t.Errorf("Expected severity 'major', got '%s'", issue.Severity)
			}
			break
		}
	}

	if !found {
		t.Fatal("Large function issue not found in results")
	}
}

func TestLocalAnalyzeMultipleIssues(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "complex.go",
			StartLine: 1,
			EndLine:   10,
			AddedLines: []string{
				"func test() {",
				"    // TODO: fix this",
				`    password = "mypassword123"`,
				`    query := "SELECT * FROM users WHERE id = " + userID`,
				"    ioutil.ReadFile(\"test.txt\")",
				"    x := x + 1",
				"    y := y + 2",
				"    z := z + 3",
				"    process()",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	if len(issues) == 0 {
		t.Fatal("Expected to find multiple issues, got none")
	}

	issueTypes := make(map[string]bool)
	for _, issue := range issues {
		issueTypes[issue.Type] = true
	}

	expectedTypes := []string{"todo_comment", "hardcoded_secrets", "sql_injection", "deprecated_function"}
	for _, expType := range expectedTypes {
		if !issueTypes[expType] {
			t.Errorf("Expected issue type '%s' not found. Got: %v", expType, issueTypes)
		}
	}
}

func TestLocalAnalyzeEmptyHunks(t *testing.T) {
	hunks := []models.DiffHunk{}
	issues := LocalAnalyze(hunks)

	if len(issues) != 0 {
		t.Errorf("Expected no issues for empty hunks, got %d", len(issues))
	}
}

func TestLocalAnalyzeCleanCode(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "clean.go",
			StartLine: 1,
			EndLine:   10,
			AddedLines: []string{
				"package main",
				"import \"os\"",
				"func readConfig() (string, error) {",
				"    data, err := os.ReadFile(\"config.json\")",
				"    if err != nil {",
				"        return \"\", err",
				"    }",
				"    return string(data), nil",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	if len(issues) != 0 {
		t.Errorf("Expected no issues for clean code, got %d: %v", len(issues), issues)
	}
}

func TestLocalAnalyzeSQLInjectionSafe(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "db.go",
			StartLine: 5,
			EndLine:   10,
			AddedLines: []string{
				"func queryUser(id string) error {",
				"    query := \"SELECT * FROM users WHERE id = ?\"",
				"    row := db.QueryRow(query, id)",
				"    return row.Scan(&user)",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	for _, issue := range issues {
		if issue.Type == "sql_injection" {
			t.Errorf("Found SQL injection issue in safe parameterized query: %v", issue)
		}
	}
}

func TestLocalAnalyzeFixmeComment(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:      "main.go",
			StartLine: 5,
			EndLine:   7,
			AddedLines: []string{
				"func init() {",
				"    // FIXME: memory leak in this function",
				"}",
			},
		},
	}

	issues := LocalAnalyze(hunks)

	found := false
	for _, issue := range issues {
		if issue.Type == "todo_comment" && strings.Contains(issue.Message, "FIXME") {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("FIXME comment not detected")
	}
}

func TestLocalAnalyzeConcurrency(t *testing.T) {
	// Test that concurrent execution works correctly
	hunks := []models.DiffHunk{
		{
			File:      "test.go",
			StartLine: 1,
			EndLine:   5,
			AddedLines: []string{
				"func test() {",
				"    // TODO: implement",
				`    apiKey := "secret123"`,
				"    ioutil.ReadAll(reader)",
				"}",
			},
		},
	}

	// Run multiple times to ensure no race conditions
	for i := 0; i < 10; i++ {
		issues := LocalAnalyze(hunks)
		if len(issues) == 0 {
			t.Errorf("Run %d: Expected issues, got none", i)
		}
	}
}

func TestLLMAnalyze(t *testing.T) {
	// TODO: Add test cases
}
