package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// PullRequestEvent represents a GitHub webhook payload for pull request events.
type PullRequestEvent struct {
	Action      string      `json:"action"`
	Number      int         `json:"number"`
	PullRequest PullRequest `json:"pull_request"`
	Repository  Repository  `json:"repository"`
}

// PullRequest contains PR details from the webhook.
type PullRequest struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	Head    Ref    `json:"head"`
	Base    Ref    `json:"base"`
	Body    string `json:"body"`
}

// Ref represents a git reference (branch/tag).
type Ref struct {
	SHA  string  `json:"sha"`
	Ref  string  `json:"ref"`
	Repo RefRepo `json:"repo"`
}

// RefRepo contains repository info for a ref.
type RefRepo struct {
	FullName string `json:"full_name"`
	CloneURL string `json:"clone_url"`
}

// Repository contains repository metadata.
type Repository struct {
	ID       int    `json:"id"`
	FullName string `json:"full_name"`
	HTMLURL  string `json:"html_url"`
}

// ValidateSignature verifies the X-Hub-Signature-256 header from GitHub.
// Returns nil if valid, error otherwise.
func ValidateSignature(body []byte, signature, secret string) error {
	if signature == "" {
		return fmt.Errorf("signature is empty")
	}

	if !strings.HasPrefix(signature, "sha256=") {
		return fmt.Errorf("signature does not start with 'sha256='")
	}

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	expectedSig := "sha256=" + hex.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ParseWebhookEvent parses the JSON body into a PullRequestEvent.
// Returns error if JSON is invalid or action is empty.
func ParseWebhookEvent(body []byte) (*PullRequestEvent, error) {
	var event PullRequestEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	if event.Action == "" {
		return nil, fmt.Errorf("action field is empty (not a valid PR event)")
	}

	return &event, nil
}

// IsAnalyzableEvent returns true if the event action triggers an analysis.
// Analyzable actions: "opened", "synchronize", "reopened"
func IsAnalyzableEvent(event *PullRequestEvent) bool {
	switch event.Action {
	case "opened", "synchronize", "reopened":
		return true
	default:
		return false
	}
}
