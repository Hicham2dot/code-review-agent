---
name: mvp-phases-tracking
description: Suivi des 11 phases d'implémentation du MVP SonarCloud-like sur Fly.io
metadata:
  type: project
---

# MVP : Code Review Agent → SonarCloud-like (Fly.io)

## 📋 Phases d'Implémentation

### ✅ Phase 0 : Nettoyage
- Suppression des fichiers Railway.app et documentation obsolète
- Conservé: code d'analyse, tests, Dockerfile, docker-compose
- **Status**: COMPLÉTÉ ✓

### ✅ Phase 1 : Dépendances (5 min)
```bash
go get github.com/go-chi/chi/v5 modernc.org/sqlite
```
- `chi` : routeur HTTP léger (v5.2.5)
- `modernc.org/sqlite` : SQLite pure Go (v1.50.1, CGO_ENABLED=0 conservé)
- Go version: 1.25.0 (mis à jour automatiquement)
- **Status**: COMPLÉTÉ ✓

### ⏳ Phase 2 : Storage SQLite
**Nouveau fichier** : `internal/storage/sqlite.go`
- Tables: repositories, analyses, issues
- Fonctions CRUD: Migrate, UpsertRepository, CreateAnalysis, UpdateAnalysisResult, ListAnalysesForRepo
- Connexion WAL mode avec timeout
 Le rôle du package storage est de :
  créer la base de données,
  créer les tables,
  enregistrer les analyses,
  enregistrer les problèmes détectés,
  récupérer les anciennes analyses.
- **Status**:  COMPLÉTÉ ✓

### ⏳ Phase 3 : GitHub Webhook Parser
**Nouveau fichier** : `internal/github/webhook.go`
- HMAC-SHA256 validation
- ParseWebhookEvent
- IsAnalyzableEvent (opened, synchronize, reopened)
- **Status**: À FAIRE

### ⏳ Phase 4 : GitHub API Client
**Nouveau fichier** : `internal/github/client.go`
- GetPRDiff(owner, repo, prNumber)
- PostPRComment(owner, repo, prNumber, body)
- CreateCheckRun(owner, repo, sha, status, conclusion, summary)
- Support PAT et GitHub App JWT (RS256)
- **Status**: À FAIRE

### ⏳ Phase 5 : Serveur HTTP
**Nouveau fichier** : `internal/server/server.go` + `internal/server/handlers.go`
- Router chi avec 7 routes
- Webhook handler avec flux async (validation → 200 → goroutine → analyse → comment PR)
- **Status**: À FAIRE

### ⏳ Phase 6 : Dashboard HTML
**Nouveau fichier** : `web/templates/dashboard.html`
- Vanilla HTML/CSS/JS (pas de framework)
- Embarqué via `//go:embed`
- Cards repos, timeline PRs, table issues filtrable
- Auto-refresh 30s
- **Status**: À FAIRE

### ⏳ Phase 7 : Commande Serve
**Nouveau fichier** : `cmd/serve.go`
- Cobra command avec flags: --port, --db, --webhook-secret, --github-app-id
- **Modification** `cmd/main.go`: +2 lignes (register serve)
- **Status**: À FAIRE

### ⏳ Phase 8 : Config Extension
**Modification** `internal/config/config.go`
- Ajout ServerConfig struct (Port, WebhookSecret, DBPath, GithubAppID, GithubKeyB64)
- Env vars: SERVER_PORT, WEBHOOK_SECRET, DB_PATH, GITHUB_APP_ID, GITHUB_KEY_B64
- **Status**: À FAIRE

### ⏳ Phase 9 : Docker Update
**Modifications** `Dockerfile`
- RUN mkdir -p /data
- EXPOSE 8080
- CMD ["serve", "--port=8080", "--db=/data/reviews.db"]

**Modification** `docker-compose.yml`
- Service: code-review-server, ports 8080, volumes sqlite_data:/data
- Secrets: NVIDIA_API_KEY, WEBHOOK_SECRET, GITHUB_APP_ID, GITHUB_KEY_B64
- **Status**: À FAIRE

### ⏳ Phase 10 : Tests End-to-End
- Build: `go build -o code-review-agent ./cmd`
- Test local: `./code-review-agent serve --port=8080 --db=:memory:`
- Tester API: `/health`, `/api/v1/analyze`, `/api/v1/repos`, etc.
- Tester dashboard: `open http://localhost:8080`
- Test webhook avec ngrok
- **Status**: À FAIRE

### ⏳ Phase 11 : Déploiement Fly.io
- `flyctl launch --no-deploy --name code-review-agent`
- `flyctl volumes create sqlite_data --size 10`
- `flyctl secrets set` (NVIDIA_API_KEY, WEBHOOK_SECRET, GITHUB_APP_ID, GITHUB_KEY_B64, GITHUB_INSTALLATION_ID)
- `flyctl deploy --push`
- Webhook URL final: `https://code-review-agent.fly.dev/webhook/github`
- Dashboard: `https://code-review-agent.fly.dev/`
- Configurer GitHub App (permissions, events, webhook URL)
- **Status**: À FAIRE

---

## 🎯 Récapitulatif

| Fichiers | Créer (7) | Modifier (3) |
|----------|-----------|-------------|
| **Nouveaux** | storage/sqlite.go, github/webhook.go, github/client.go, server/server.go, server/handlers.go, cmd/serve.go, web/templates/dashboard.html | |
| **Existants** | | cmd/main.go (+2), config/config.go (+ServerConfig), Dockerfile (+3), docker-compose.yml (+env) |

## 🚀 Aucune modification aux fonctions d'analyse existantes
LocalAnalyze, LLMAnalyze, Aggregate, FormatMarkdown sont réutilisées telles quelles.
