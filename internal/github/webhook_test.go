package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
)

func TestValidateSignature_Valid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action": "opened"}`)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)
	signature := "sha256=" + hex.EncodeToString(h.Sum(nil))

	err := ValidateSignature(body, signature, secret)
	if err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}
}

func TestValidateSignature_Invalid(t *testing.T) {
	secret := "test-secret"
	body := []byte(`{"action": "opened"}`)
	signature := "sha256=0000000000000000000000000000000000000000000000000000000000000000"

	err := ValidateSignature(body, signature, secret)
	if err == nil {
		t.Error("expected error for invalid signature, got nil")
	}

	if err.Error() != "signature mismatch" {
		t.Errorf("expected 'signature mismatch', got: %v", err)
	}
}

func TestValidateSignature_Empty(t *testing.T) {
	body := []byte(`{"action": "opened"}`)

	err := ValidateSignature(body, "", "secret")
	if err == nil {
		t.Error("expected error for empty signature, got nil")
	}

	if err.Error() != "signature is empty" {
		t.Errorf("expected 'signature is empty', got: %v", err)
	}
}

func TestValidateSignature_NoPrefix(t *testing.T) {
	body := []byte(`{"action": "opened"}`)
	signature := "0000000000000000000000000000000000000000000000000000000000000000"

	err := ValidateSignature(body, signature, "secret")
	if err == nil {
		t.Error("expected error for missing sha256= prefix, got nil")
	}

	if err.Error() != "signature does not start with 'sha256='" {
		t.Errorf("expected 'signature does not start with sha256=', got: %v", err)
	}
}

func TestParseWebhookEvent_Valid(t *testing.T) {
	payload := map[string]interface{}{
		"action": "opened",
		"number": 42,
		"pull_request": map[string]interface{}{
			"id":       1,
			"title":    "Fix bug",
			"state":    "open",
			"html_url": "https://github.com/user/repo/pull/42",
			"head": map[string]interface{}{
				"sha": "abc123",
				"ref": "feature/fix",
				"repo": map[string]interface{}{
					"full_name": "user/repo",
					"clone_url": "https://github.com/user/repo.git",
				},
			},
			"base": map[string]interface{}{
				"sha": "def456",
				"ref": "main",
				"repo": map[string]interface{}{
					"full_name": "user/repo",
					"clone_url": "https://github.com/user/repo.git",
				},
			},
			"body": "This PR fixes issue #123",
		},
		"repository": map[string]interface{}{
			"id":        12345,
			"full_name": "user/repo",
			"html_url":  "https://github.com/user/repo",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	event, err := ParseWebhookEvent(body)
	if err != nil {
		t.Errorf("expected nil, got error: %v", err)
	}

	if event.Action != "opened" {
		t.Errorf("expected action 'opened', got '%s'", event.Action)
	}

	if event.Number != 42 {
		t.Errorf("expected number 42, got %d", event.Number)
	}

	if event.PullRequest.Title != "Fix bug" {
		t.Errorf("expected title 'Fix bug', got '%s'", event.PullRequest.Title)
	}

	if event.Repository.FullName != "user/repo" {
		t.Errorf("expected repo 'user/repo', got '%s'", event.Repository.FullName)
	}
}

func TestParseWebhookEvent_Invalid(t *testing.T) {
	body := []byte(`{invalid json`)

	event, err := ParseWebhookEvent(body)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}

	if event != nil {
		t.Errorf("expected nil event, got %+v", event)
	}
}

func TestParseWebhookEvent_EmptyAction(t *testing.T) {
	payload := map[string]interface{}{
		"action": "",
		"number": 1,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal test payload: %v", err)
	}

	event, err := ParseWebhookEvent(body)
	if err == nil {
		t.Error("expected error for empty action, got nil")
	}

	if event != nil {
		t.Errorf("expected nil event, got %+v", event)
	}
}

func TestIsAnalyzableEvent_Opened(t *testing.T) {
	event := &PullRequestEvent{Action: "opened"}
	if !IsAnalyzableEvent(event) {
		t.Error("expected 'opened' to be analyzable")
	}
}

func TestIsAnalyzableEvent_Synchronize(t *testing.T) {
	event := &PullRequestEvent{Action: "synchronize"}
	if !IsAnalyzableEvent(event) {
		t.Error("expected 'synchronize' to be analyzable")
	}
}

func TestIsAnalyzableEvent_Reopened(t *testing.T) {
	event := &PullRequestEvent{Action: "reopened"}
	if !IsAnalyzableEvent(event) {
		t.Error("expected 'reopened' to be analyzable")
	}
}

func TestIsAnalyzableEvent_Closed(t *testing.T) {
	event := &PullRequestEvent{Action: "closed"}
	if IsAnalyzableEvent(event) {
		t.Error("expected 'closed' to NOT be analyzable")
	}
}

func TestIsAnalyzableEvent_Labeled(t *testing.T) {
	event := &PullRequestEvent{Action: "labeled"}
	if IsAnalyzableEvent(event) {
		t.Error("expected 'labeled' to NOT be analyzable")
	}
}

func TestIsAnalyzableEvent_Edited(t *testing.T) {
	event := &PullRequestEvent{Action: "edited"}
	if IsAnalyzableEvent(event) {
		t.Error("expected 'edited' to NOT be analyzable")
	}
}
