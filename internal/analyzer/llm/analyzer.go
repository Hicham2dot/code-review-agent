package llm

import (
	"bytes"
	"code-review-agent/internal/config"
	"code-review-agent/internal/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	mistralAPIBase = "https://api.mistral.ai/v1/chat/completions"
	systemPrompt   = `Tu es un expert en code review et en sécurité. Analyse le diff fourni et retourne UNIQUEMENT un tableau JSON valide.
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

type messageRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []messageItem `json:"messages"`
}

type messageItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type mistralChoice struct {
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
}

type messageResponse struct {
	Choices []mistralChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type hfResponse []struct {
	GeneratedText string `json:"generated_text"`
}

func LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error) {
	apiKey := os.Getenv("MISTRAL_API_KEY")
	if apiKey == "" {
		return []models.Issue{}, fmt.Errorf("MISTRAL_API_KEY not set")
	}

	model := cfg.Model
	if model == "" {
		model = "mistral-medium"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens == 0 {
		maxTokens = 1024
	}

	prompt := BuildPrompt(hunks)

	reqBody := messageRequest{
		Model:     model,
		MaxTokens: maxTokens,
		Messages: []messageItem{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", mistralAPIBase, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	var apiResp messageResponse
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

	if len(apiResp.Choices) == 0 {
		return []models.Issue{}, nil
	}

	text := apiResp.Choices[0].Message.Content
	issues := ParseLLMResponse(text)

	return issues, nil
}
