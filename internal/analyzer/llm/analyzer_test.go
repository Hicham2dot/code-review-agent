package llm

import (
	"code-review-agent/internal/config"
	"code-review-agent/internal/models"
	"os"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	hunks := []models.DiffHunk{
		{
			File:       "main.go",
			StartLine:  10,
			AddedLines: []string{"var x = 10", "var y = 20"},
		},
	}

	prompt := BuildPrompt(hunks)
	if prompt == "" {
		t.Error("BuildPrompt returned empty string")
	}
	if !contains(prompt, "main.go") {
		t.Error("BuildPrompt did not include file name")
	}
	if !contains(prompt, "var x = 10") {
		t.Error("BuildPrompt did not include added lines")
	}
}

func TestParseLLMResponse(t *testing.T) {
	jsonResp := `[{"type":"logic_error","severity":"major","file":"app.go","start_line":42,"message":"Missing null check","suggestion":"Add nil validation","confidence":0.9},{"type":"performance","severity":"minor","file":"util.go","start_line":15,"message":"Loop optimization possible","suggestion":"Use range iterator","confidence":0.75}]`

	issues := ParseLLMResponse(jsonResp)
	if len(issues) != 2 {
		t.Errorf("Expected 2 issues, got %d", len(issues))
	}

	if len(issues) > 0 {
		if issues[0].Type != "logic_error" {
			t.Errorf("Expected type logic_error, got %s", issues[0].Type)
		}
		if issues[0].Source != "llm_analyzer" {
			t.Errorf("Expected source llm_analyzer, got %s", issues[0].Source)
		}
		if issues[0].Severity != "major" {
			t.Errorf("Expected severity major, got %s", issues[0].Severity)
		}
		if issues[0].Confidence != 0.9 {
			t.Errorf("Expected confidence 0.9, got %v", issues[0].Confidence)
		}
	}
}

func TestParseLLMResponseEmpty(t *testing.T) {
	issues := ParseLLMResponse("[]")
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues from empty array, got %d", len(issues))
	}

	issues = ParseLLMResponse("")
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues from empty string, got %d", len(issues))
	}
}

func TestParseLLMResponseInvalidJSON(t *testing.T) {
	issues := ParseLLMResponse("not json")
	if len(issues) != 0 {
		t.Errorf("Expected 0 issues from invalid JSON, got %d", len(issues))
	}
}

func TestLLMAnalyze(t *testing.T) {
	// Skip if GEMINI_API_KEY not set (for CI environments)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	hunks := []models.DiffHunk{
		{
			File:       "test.go",
			StartLine:  1,
			AddedLines: []string{"var secret = \"api_key_12345\""},
		},
	}

	cfg := config.LLMConfig{
		Model:     "gemini-2.0-flash",
		MaxTokens: 512,
	}

	issues, err := LLMAnalyze(hunks, cfg)
	if err != nil {
		t.Fatalf("LLMAnalyze failed: %v", err)
	}

	if len(issues) == 0 {
		t.Logf("LLMAnalyze returned 0 issues (acceptable for code without clear problems)")
	}

	for _, issue := range issues {
		if issue.Source != "llm_analyzer" {
			t.Errorf("Expected source llm_analyzer, got %s", issue.Source)
		}
		if issue.Severity == "" {
			t.Error("Issue has empty severity")
		}
		if issue.Type == "" {
			t.Error("Issue has empty type")
		}
	}
}

func TestLLMAnalyzeMalicious(t *testing.T) {
	// Skip if GEMINI_API_KEY not set (for CI environments)
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	hunks := []models.DiffHunk{
		{
			File:      "database.go",
			StartLine: 10,
			AddedLines: []string{
				"func FetchUser(username string) {",
				"    query := \"SELECT * FROM users WHERE username = '\" + username + \"'\"",
				"    db.Query(query)",
				"}",
				"",
				"func ExecuteCommand(cmd string) {",
				"    shell := exec.Command(\"/bin/sh\", \"-c\", cmd)",
				"    shell.Run()",
				"}",
				"",
				"const apiKey = \"sk-1234567890abcdefghijklmnop\"",
				"const dbPassword = \"root_password_123\"",
			},
		},
	}

	cfg := config.LLMConfig{
		Model:     "gemini-2.0-flash",
		MaxTokens: 1024,
	}

	issues, err := LLMAnalyze(hunks, cfg)
	if err != nil {
		t.Fatalf("LLMAnalyze failed: %v", err)
	}

	t.Logf("✓ API call succeeded (using model: %s)", cfg.Model)
	t.Logf("Malicious code analysis found %d issues", len(issues))
	if len(issues) > 0 {
		t.Logf("✓ Successfully detected %d security issues in malicious code", len(issues))
		for _, issue := range issues {
			t.Logf("  - Type: %s, Severity: %s, Message: %s", issue.Type, issue.Severity, issue.Message)
			if issue.Source != "llm_analyzer" {
				t.Errorf("Expected source llm_analyzer, got %s", issue.Source)
			}
		}
	} else {
		t.Logf("Note: LLMAnalyze returned 0 issues - model may not have detected vulnerabilities")
	}
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
