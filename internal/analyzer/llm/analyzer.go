package llm

import (
	"bytes"
	"code-review-agent/internal/config"
	"code-review-agent/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

const (
	geminiAPIBase = "https://generativelanguage.googleapis.com/v1/models/%s:generateContent"
	systemPrompt  = `Tu es un expert en code review et en sécurité. Analyse le diff fourni et retourne UNIQUEMENT un tableau JSON valide.
Chaque problème détecté doit avoir la structure suivante :
[{"type":"...","severity":"critical|major|minor","file":"...","start_line":N,"message":"...","suggestion":"...","confidence":0.0-1.0}]
Détecte TOUS les problèmes :
- Sécurité: hardcoded secrets/API keys/passwords, SQL injection, command injection, XXE, CSRF, weak crypto
- Logique: null checks manquants, race conditions, logic errors
- Performance: inefficiencies, memory leaks, N+1 queries
- Code quality: code non idiomatique, duplication, antipatterns
Si aucun problème, retourne [].
Réponds UNIQUEMENT avec le JSON, aucun texte avant ou après.`
)

type geminiRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		MaxOutputTokens int `json:"maxOutputTokens"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return []models.Issue{}, fmt.Errorf("GEMINI_API_KEY not set")
	}

	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	prompt := BuildPrompt(hunks)
	fullPrompt := systemPrompt + "\n\n" + prompt

	reqBody := geminiRequest{}
	reqBody.Contents = []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		{
			Role: "user",
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: fullPrompt},
			},
		},
	}
	reqBody.GenerationConfig.MaxOutputTokens = maxTokens

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	apiURL := fmt.Sprintf(geminiAPIBase, url.QueryEscape(model))
	req, err := http.NewRequest("POST", apiURL+"?key="+apiKey, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var apiResp geminiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		truncated := string(respBody)
		if len(truncated) > 200 {
			truncated = truncated[:200]
		}
		return nil, fmt.Errorf("parse response: %w (body: %s)", err, truncated)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Candidates) == 0 {
		return []models.Issue{}, nil
	}

	text := apiResp.Candidates[0].Content.Parts[0].Text
	issues := ParseLLMResponse(text)

	return issues, nil
}
