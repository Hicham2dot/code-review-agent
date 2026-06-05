package github

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"code-review-agent/internal/config"
)

const defaultBaseURL = "https://api.github.com"

// Client interacts with the GitHub REST API.
type Client struct {
	httpClient *http.Client
	cfg        config.GitHubConfig
	baseURL    string
}

// NewClient creates a GitHub API client from the given config.
func NewClient(cfg config.GitHubConfig) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cfg:        cfg,
		baseURL:    defaultBaseURL,
	}
}

// getToken returns a Bearer token for API calls.
// Uses PAT directly, or generates a GitHub App installation token via JWT exchange.
func (c *Client) getToken() (string, error) {
	if c.cfg.Token != "" {
		return c.cfg.Token, nil
	}
	hasKey := c.cfg.PrivateKeyPath != "" || c.cfg.PrivateKeyB64 != ""
	if c.cfg.AppID != "" && hasKey && c.cfg.InstallationID != "" {
		return c.appInstallationToken()
	}
	return "", fmt.Errorf("no GitHub credentials configured: set GITHUB_TOKEN or GITHUB_APP_ID + (GITHUB_KEY_B64 or GITHUB_PRIVATE_KEY_PATH) + GITHUB_INSTALLATION_ID")
}

// appInstallationToken generates a GitHub App JWT and exchanges it for an installation access token.
func (c *Client) appInstallationToken() (string, error) {
	jwt, err := c.buildAppJWT()
	if err != nil {
		return "", fmt.Errorf("build app JWT: %w", err)
	}

	url := fmt.Sprintf("%s/app/installations/%s/access_tokens", c.baseURL, c.cfg.InstallationID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("installation token request failed (%d): %s", resp.StatusCode, body)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parse installation token response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("installation token response contained empty token")
	}
	return result.Token, nil
}

// buildAppJWT constructs a signed RS256 JWT for GitHub App authentication.
// JWT is valid for 9 minutes (GitHub maximum is 10).
func (c *Client) buildAppJWT() (string, error) {
	var keyData []byte
	if c.cfg.PrivateKeyB64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(c.cfg.PrivateKeyB64)
		if err != nil {
			return "", fmt.Errorf("decode base64 private key: %w", err)
		}
		keyData = decoded
	} else {
		data, err := os.ReadFile(c.cfg.PrivateKeyPath)
		if err != nil {
			return "", fmt.Errorf("read private key: %w", err)
		}
		keyData = data
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block from private key file")
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format as fallback
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return "", fmt.Errorf("parse RSA private key (PKCS1: %v, PKCS8: %v)", err, err2)
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("private key is not an RSA key")
		}
	}

	now := time.Now()
	header := base64url([]byte(`{"alg":"PS256","typ":"JWT"}`))
	payload := base64url([]byte(fmt.Sprintf(
		`{"iss":"%s","iat":%d,"exp":%d}`,
		c.cfg.AppID,
		now.Unix(),
		now.Add(9*time.Minute).Unix(),
	)))

	signingInput := header + "." + payload
	h := sha256.New()
	h.Write([]byte(signingInput))
	digest := h.Sum(nil)

	sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	})
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signingInput + "." + base64url(sig), nil
}

// base64url encodes bytes using base64 URL encoding without padding.
func base64url(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// doRequest performs an authenticated HTTP request and returns the response body.
// Caller is responsible for closing the body only if no error is returned and body is non-nil.
func (c *Client) doRequest(method, url string, body io.Reader, accept string) (*http.Response, error) {
	token, err := c.getToken()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", accept)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

// GetPRDiff fetches the unified diff for the given pull request number.
// The returned string is ready to pass into parser.ParseDiff().
func (c *Client) GetPRDiff(owner, repo string, prNumber int) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, prNumber)

	resp, err := c.doRequest("GET", url, nil, "application/vnd.github.v3.diff")
	if err != nil {
		return "", fmt.Errorf("GetPRDiff request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GetPRDiff failed (%d): %s", resp.StatusCode, body)
	}

	diff, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read diff body: %w", err)
	}
	return string(diff), nil
}

// PostPRComment posts a Markdown comment on the given pull request.
// Use formatter.FormatMarkdown() to produce the body string.
func (c *Client) PostPRComment(owner, repo string, prNumber int, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", c.baseURL, owner, repo, prNumber)

	payload, err := json.Marshal(map[string]string{"body": body})
	if err != nil {
		return err
	}

	resp, err := c.doRequest("POST", url, bytes.NewReader(payload), "application/vnd.github.v3+json")
	if err != nil {
		return fmt.Errorf("PostPRComment request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("PostPRComment failed (%d): %s", resp.StatusCode, respBody)
	}
	return nil
}

// CheckRunOutput holds the user-visible content for a GitHub Check Run.
type CheckRunOutput struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// CreateCheckRun creates a GitHub Check Run on the given commit SHA.
// status must be one of: "queued", "in_progress", "completed".
// conclusion must be one of: "success", "failure", "neutral" (required when status="completed").
// Requires a GitHub App token — PAT tokens cannot create check runs.
func (c *Client) CreateCheckRun(owner, repo, sha, status, conclusion, summary string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/check-runs", c.baseURL, owner, repo)

	type checkRunRequest struct {
		Name       string         `json:"name"`
		HeadSHA    string         `json:"head_sha"`
		Status     string         `json:"status"`
		Conclusion string         `json:"conclusion,omitempty"`
		Output     CheckRunOutput `json:"output"`
	}

	reqBody := checkRunRequest{
		Name:       "code-review-agent",
		HeadSHA:    sha,
		Status:     status,
		Conclusion: conclusion,
		Output: CheckRunOutput{
			Title:   "Code Review Analysis",
			Summary: summary,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	resp, err := c.doRequest("POST", url, bytes.NewReader(payload), "application/vnd.github.v3+json")
	if err != nil {
		return fmt.Errorf("CreateCheckRun request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("CreateCheckRun failed (%d): %s", resp.StatusCode, respBody)
	}
	return nil
}
