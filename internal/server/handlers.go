package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"code-review-agent/internal/aggregator"
	"code-review-agent/internal/analyzer/llm"
	"code-review-agent/internal/analyzer/local"
	"code-review-agent/internal/cache"
	"code-review-agent/internal/formatter"
	"code-review-agent/internal/github"
	"code-review-agent/internal/models"
	"code-review-agent/internal/parser"
	web "code-review-agent/web"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

// GET /health
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GET /api/v1/status
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	uptime := int64(time.Since(s.startAt).Seconds())

	var analysesCount int
	if s.store != nil {
		if list, err := s.store.ListRecentAnalyses(1000); err == nil {
			analysesCount = len(list)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "ok",
		"uptime_seconds": uptime,
		"version":        "0.1.0",
		"analyses_count": analysesCount,
		"llm_enabled":    s.cfg.Analysis.AIEnabled,
		"cache_enabled":  s.cfg.Cache.Enabled,
	})
}

// POST /webhook  — async: validate → 200 → goroutine → analyse → comment PR
func (s *Server) webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	sig := r.Header.Get("X-Hub-Signature-256")
	if s.cfg.Server.WebhookSecret != "" {
		if err := github.ValidateSignature(body, sig, s.cfg.Server.WebhookSecret); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
			return
		}
	}

	event, err := github.ParseWebhookEvent(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if !github.IsAnalyzableEvent(event) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "action": event.Action})
		return
	}

	// Respond immediately, process in background
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "accepted",
		"pr":     fmt.Sprintf("%s#%d", event.Repository.FullName, event.Number),
	})

	go s.processWebhookAsync(event)
}

func (s *Server) processWebhookAsync(event *github.PullRequestEvent) {
	parts := strings.SplitN(event.Repository.FullName, "/", 2)
	if len(parts) != 2 {
		log.Printf("[webhook] invalid repo full_name: %s", event.Repository.FullName)
		return
	}
	owner, repo := parts[0], parts[1]
	prNumber := event.Number

	ghClient := github.NewClient(s.cfg.GitHub)

	diff, err := ghClient.GetPRDiff(owner, repo, prNumber)
	if err != nil {
		log.Printf("[webhook] GetPRDiff %s#%d: %v", event.Repository.FullName, prNumber, err)
		return
	}

	hunks := parser.ParseDiff(diff)

	localIssues := local.LocalAnalyze(hunks)

	var llmIssues []models.Issue
	if s.cfg.Analysis.AIEnabled {
		if li, err := llm.LLMAnalyze(hunks, s.cfg.LLM); err == nil {
			llmIssues = li
		} else {
			log.Printf("[webhook] LLMAnalyze: %v", err)
		}
	}

	result := aggregator.Aggregate(localIssues, llmIssues, hunks, diff)

	comment := formatter.FormatMarkdown(&result)
	if err := ghClient.PostPRComment(owner, repo, prNumber, comment); err != nil {
		log.Printf("[webhook] PostPRComment %s#%d: %v", event.Repository.FullName, prNumber, err)
	}

	if s.store != nil {
		repoID, err := s.store.UpsertRepository(repo, event.Repository.FullName)
		if err == nil {
			if _, err := s.store.CreateAnalysis(repoID, &result); err != nil {
				log.Printf("[webhook] CreateAnalysis: %v", err)
			}
		}
	}
}

// POST /api/v1/analyze  — synchronous on-demand analysis (body = raw diff)
func (s *Server) analyzeHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil || len(body) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body must contain a unified diff"})
		return
	}

	diff := string(body)
	hunks := parser.ParseDiff(diff)

	localIssues := local.LocalAnalyze(hunks)

	var llmIssues []models.Issue
	if s.cfg.Analysis.AIEnabled {
		if li, err := llm.LLMAnalyze(hunks, s.cfg.LLM); err == nil {
			llmIssues = li
		}
	}

	result := aggregator.Aggregate(localIssues, llmIssues, hunks, diff)

	writeJSON(w, http.StatusOK, &result)
}

// GET /api/v1/analyses
func (s *Server) listAnalysesHandler(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage not configured"})
		return
	}

	list, err := s.store.ListRecentAnalyses(100)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if list == nil {
		list = []models.AnalysisResult{}
	}
	writeJSON(w, http.StatusOK, list)
}

// GET /api/v1/analyses/{hash}
func (s *Server) getAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hash is required"})
		return
	}

	if s.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage not configured"})
		return
	}

	result, err := s.store.GetAnalysisByHash(hash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if result == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "analysis not found"})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// DELETE /api/v1/cache
func (s *Server) clearCacheHandler(w http.ResponseWriter, r *http.Request) {
	if err := cache.Clear(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cache cleared"})
}

// GET /dashboard — sert le fichier HTML embarqué via go:embed
func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	data, err := web.FS.ReadFile("templates/dashboard.html")
	if err != nil {
		http.Error(w, "dashboard not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(data) //nolint:errcheck
}

// GET /api/v1/analyses/{hash}/issues
func (s *Server) getAnalysisIssuesHandler(w http.ResponseWriter, r *http.Request) {
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hash is required"})
		return
	}

	if s.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage not configured"})
		return
	}

	issues, err := s.store.GetIssuesByAnalysisHash(hash)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, issues)
}
