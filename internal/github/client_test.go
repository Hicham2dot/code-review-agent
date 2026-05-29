package github

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"code-review-agent/internal/config"
)

// newTestClient returns a Client pointed at the mock server's URL.
func newTestClient(t *testing.T, cfg config.GitHubConfig, serverURL string) *Client {
	t.Helper()
	c := NewClient(cfg)
	c.baseURL = serverURL
	return c
}

// writeTempPrivateKey generates an RSA key and writes it as a PEM file, returning the path.
func writeTempPrivateKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	keyBytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: keyBytes}

	path := filepath.Join(t.TempDir(), "test.pem")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create key file: %v", err)
	}
	defer f.Close()
	if err := pem.Encode(f, block); err != nil {
		t.Fatalf("encode PEM: %v", err)
	}
	return path
}

// decodeBase64URL decodes a base64url-encoded string (no padding required).
func decodeBase64URL(s string) ([]byte, error) {
	// Normalize base64url → base64
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	// Re-add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.StdEncoding.DecodeString(s)
}

// --- Auth tests ---

func TestGetToken_PAT(t *testing.T) {
	c := &Client{cfg: config.GitHubConfig{Token: "ghp_testtoken"}}
	tok, err := c.getToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "ghp_testtoken" {
		t.Errorf("expected ghp_testtoken, got %q", tok)
	}
}

func TestGetToken_NoCredentials(t *testing.T) {
	c := &Client{cfg: config.GitHubConfig{}}
	_, err := c.getToken()
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
}

func TestBuildAppJWT(t *testing.T) {
	keyPath := writeTempPrivateKey(t)
	c := &Client{cfg: config.GitHubConfig{
		AppID:          "12345",
		PrivateKeyPath: keyPath,
		InstallationID: "99",
	}}

	jwt, err := c.buildAppJWT()
	if err != nil {
		t.Fatalf("buildAppJWT: %v", err)
	}

	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT parts, got %d", len(parts))
	}

	headerJSON, err := decodeBase64URL(parts[0])
	if err != nil {
		t.Fatalf("decode JWT header: %v", err)
	}
	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		t.Fatalf("unmarshal JWT header: %v", err)
	}
	if header["alg"] != "RS256" {
		t.Errorf("expected alg RS256, got %q", header["alg"])
	}
	if header["typ"] != "JWT" {
		t.Errorf("expected typ JWT, got %q", header["typ"])
	}

	payloadJSON, err := decodeBase64URL(parts[1])
	if err != nil {
		t.Fatalf("decode JWT payload: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal JWT payload: %v", err)
	}
	if payload["iss"] != "12345" {
		t.Errorf("expected iss 12345, got %v", payload["iss"])
	}
}

func TestAppInstallationToken(t *testing.T) {
	keyPath := writeTempPrivateKey(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/app/installations/99/access_tokens" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token, got %q", auth)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"token": "ghs_installation_token"})
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{
		AppID:          "42",
		PrivateKeyPath: keyPath,
		InstallationID: "99",
	}, srv.URL)

	tok, err := c.getToken()
	if err != nil {
		t.Fatalf("getToken via App JWT: %v", err)
	}
	if tok != "ghs_installation_token" {
		t.Errorf("expected ghs_installation_token, got %q", tok)
	}
}

// --- GetPRDiff tests ---

func TestGetPRDiff_Success(t *testing.T) {
	const wantDiff = "diff --git a/file.go b/file.go\n+++ added line\n"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/pulls/42" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3.diff" {
			t.Errorf("expected diff Accept header, got %q", r.Header.Get("Accept"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(wantDiff))
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	diff, err := c.GetPRDiff("owner", "repo", 42)
	if err != nil {
		t.Fatalf("GetPRDiff: %v", err)
	}
	if diff != wantDiff {
		t.Errorf("diff mismatch: got %q", diff)
	}
}

func TestGetPRDiff_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	_, err := c.GetPRDiff("owner", "repo", 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

// --- PostPRComment tests ---

func TestPostPRComment_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/issues/7/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["body"] != "## Analysis\nno issues found" {
			t.Errorf("unexpected body field: %q", body["body"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": 1})
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	err := c.PostPRComment("owner", "repo", 7, "## Analysis\nno issues found")
	if err != nil {
		t.Fatalf("PostPRComment: %v", err)
	}
}

func TestPostPRComment_Forbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message":"Forbidden"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	err := c.PostPRComment("owner", "repo", 7, "body")
	if err == nil {
		t.Fatal("expected error for 403")
	}
}

// --- CreateCheckRun tests ---

func TestCreateCheckRun_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/check-runs" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["head_sha"] != "abc123" {
			t.Errorf("unexpected head_sha: %v", body["head_sha"])
		}
		if body["status"] != "completed" {
			t.Errorf("unexpected status: %v", body["status"])
		}
		if body["conclusion"] != "success" {
			t.Errorf("unexpected conclusion: %v", body["conclusion"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": 1})
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	err := c.CreateCheckRun("owner", "repo", "abc123", "completed", "success", "All checks passed")
	if err != nil {
		t.Fatalf("CreateCheckRun: %v", err)
	}
}

func TestCreateCheckRun_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, config.GitHubConfig{Token: "tok"}, srv.URL)
	err := c.CreateCheckRun("owner", "repo", "abc123", "completed", "failure", "analysis failed")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}
