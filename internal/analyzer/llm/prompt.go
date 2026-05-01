package llm

import (
	"code-review-agent/internal/models"
	"encoding/json"
	"fmt"
	"strings"
)

func BuildPrompt(hunks []models.DiffHunk) string {
	var sb strings.Builder
	sb.WriteString("Analyse le diff suivant et détecte les problèmes de code :\n\n")

	for _, hunk := range hunks {
		sb.WriteString(fmt.Sprintf("=== %s (ligne %d) ===\n", hunk.File, hunk.StartLine))
		for i, line := range hunk.AddedLines {
			lineNum := hunk.StartLine + i
			sb.WriteString(fmt.Sprintf("+%d: %s\n", lineNum, line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

type llmIssueResponse struct {
	Type       string  `json:"type"`
	Severity   string  `json:"severity"`
	File       string  `json:"file"`
	StartLine  int     `json:"start_line"`
	Message    string  `json:"message"`
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
}

func ParseLLMResponse(raw string) []models.Issue {
	var issues []models.Issue
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return issues
	}

	// Extract JSON from markdown code block if present
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "\n")
		if idx := strings.LastIndex(raw, "```"); idx != -1 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimPrefix(raw, "\n")
		if idx := strings.LastIndex(raw, "```"); idx != -1 {
			raw = raw[:idx]
		}
		raw = strings.TrimSpace(raw)
	}

	var responses []llmIssueResponse
	if err := json.Unmarshal([]byte(raw), &responses); err != nil {
		return issues
	}

	for _, resp := range responses {
		issues = append(issues, models.Issue{
			ID:         fmt.Sprintf("llm-%s-%d", resp.Type, resp.StartLine),
			Type:       resp.Type,
			Severity:   resp.Severity,
			Location: models.Location{
				File:      resp.File,
				StartLine: resp.StartLine,
				EndLine:   resp.StartLine,
			},
			Message:    resp.Message,
			Suggestion: resp.Suggestion,
			Confidence: resp.Confidence,
			Source:     "llm_analyzer",
		})
	}

	return issues
}
