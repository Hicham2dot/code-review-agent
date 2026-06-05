package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"

	"code-review-agent/internal/aggregator"
	"code-review-agent/internal/analyzer/llm"
	"code-review-agent/internal/analyzer/local"
	"code-review-agent/internal/cache"
	"code-review-agent/internal/formatter"
	"code-review-agent/internal/github"
	"code-review-agent/internal/models"
	"code-review-agent/internal/parser"
	"code-review-agent/internal/storage"
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

	// Persist to SQLite if store is configured
	if s.store != nil {
		repoID, err := s.store.UpsertRepository("on-demand", "on-demand")
		if err == nil {
			if _, err := s.store.CreateAnalysis(repoID, &result); err != nil {
				log.Printf("[analyze] CreateAnalysis: %v", err)
			}
		}
	}

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

// GET /dashboard — redirige vers le bon dashboard selon le rôle
func (s *Server) dashboardHandler(w http.ResponseWriter, r *http.Request) {
	role := ctxGetRole(r)
	if role == "admin" {
		s.adminDashboardHandler(w, r)
		return
	}
	s.userDashboardHandler(w, r)
}

// GET /dashboard (user) — soumettre un diff + voir ses propres analyses
func (s *Server) userDashboardHandler(w http.ResponseWriter, r *http.Request) {
	userID := ctxGetUserID(r)

	var analyses []models.AnalysisResult
	if s.store != nil {
		analyses, _ = s.store.ListAnalysesForUser(userID, 20)
	}

	username := ""
	if s.store != nil {
		// get username from DB — reuse GetUserByUsername workaround via session
	}

	renderTemplate(w, "templates/dashboard_user.html", map[string]any{
		"Username": username,
		"Analyses": analyses,
	})
}

// GET /dashboard (admin) — tous les users + toutes les analyses
func (s *Server) adminDashboardHandler(w http.ResponseWriter, r *http.Request) {
	var users []storage.UserInfo
	var analyses []models.AnalysisResult
	totalIssues, totalCritical := 0, 0

	if s.store != nil {
		users, _ = s.store.ListUsers()
		analyses, _ = s.store.ListRecentAnalyses(100)
		for _, a := range analyses {
			totalIssues += a.Summary.TotalIssues
			totalCritical += a.Summary.CriticalCount
		}
	}

	renderTemplate(w, "templates/dashboard_admin.html", map[string]any{
		"Username":      "admin",
		"TotalUsers":    len(users),
		"TotalAnalyses": len(analyses),
		"TotalIssues":   totalIssues,
		"TotalCritical": totalCritical,
		"Users":         users,
		"Analyses":      analyses,
	})
}

// GET /api/v1/admin/users — liste tous les utilisateurs (admin only)
func (s *Server) adminListUsersHandler(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage not configured"})
		return
	}
	users, err := s.store.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// POST /api/v1/admin/users — crée un nouvel utilisateur (admin only)
func (s *Server) adminCreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "storage not configured"})
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}
	if req.Role != "admin" && req.Role != "user" {
		req.Role = "user"
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 8 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	id, err := s.store.CreateUser(req.Username, string(hash), req.Role)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":       id,
		"username": req.Username,
		"role":     req.Role,
	})
}

func renderTemplate(w http.ResponseWriter, path string, data any) {
	raw, err := web.FS.ReadFile(path)
	if err != nil {
		http.Error(w, "template not found", http.StatusInternalServerError)
		return
	}
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"slice": func(s string, i, j int) string {
			if j > len(s) {
				j = len(s)
			}
			return s[i:j]
		},
	}).Parse(string(raw))
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	tmpl.Execute(w, data) //nolint:errcheck
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

// GET /login — affiche la page de connexion
func (s *Server) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	renderLogin(w, "")
}

// POST /login — valide les credentials et crée une session
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		renderLogin(w, "Requête invalide.")
		return
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	if username == "" || password == "" {
		renderLogin(w, "Nom d'utilisateur et mot de passe requis.")
		return
	}

	if s.store == nil {
		renderLogin(w, "Base de données non disponible.")
		return
	}

	userID, hash, _, err := s.store.GetUserByUsername(username)
	if err != nil || userID == 0 {
		renderLogin(w, "Identifiants incorrects.")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		renderLogin(w, "Identifiants incorrects.")
		return
	}

	token, err := generateToken()
	if err != nil {
		renderLogin(w, "Erreur interne.")
		return
	}

	expires := time.Now().Add(24 * time.Hour)
	if err := s.store.CreateSession(token, userID, expires.UTC().Format("2006-01-02 15:04:05")); err != nil {
		renderLogin(w, "Erreur interne.")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// POST /logout — supprime la session et redirige vers /login
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("session"); err == nil && s.store != nil {
		s.store.DeleteSession(cookie.Value) //nolint:errcheck
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func renderLogin(w http.ResponseWriter, errMsg string) {
	tmplData, err := web.FS.ReadFile("templates/login.html")
	if err != nil {
		http.Error(w, "login page not found", http.StatusInternalServerError)
		return
	}
	tmpl, err := template.New("login").Parse(string(tmplData))
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if errMsg != "" {
		w.WriteHeader(http.StatusUnauthorized)
	}
	tmpl.Execute(w, struct{ Error string }{Error: errMsg}) //nolint:errcheck
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
