# Code Review Agent - Documentation Interne

## 1. Objectif Principal
Outil CLI et API qui analyse automatiquement les modifications de code (diffs) via des checks statiques locaux et une analyse LLM pour détecter les problèmes de qualité, sécurité et conformité.

## 2. Règles Métier Essentielles

### Analyse Locale (6 règles)
1. **hardcoded_secrets** (critical, 0.98 confidence)
   - Détecte: API keys, passwords, tokens, DB credentials, AWS keys, private PEM keys
   - Patterns: `api_key=["']...`, `password=["']...`, `AKIA[0-9A-Z]{16}`, `-----BEGIN...PRIVATE KEY`

2. **sql_injection** (critical, 0.85 confidence)
   - Détecte: Concatenations de strings SQL, fmt.Sprintf avec SELECT, unparameterized queries
   - Safe indicators: `?`, `$1`, `$2`, `@param`, `prepared`

3. **todo_comment** (minor, 0.99 confidence)
   - Détecte: `TODO:`, `FIXME:`, `XXX:`, `HACK:`, `BUG:` dans le code

4. **large_function** (major, 0.95 confidence)
   - Seuil: > 50 lignes dans une fonction
   - Tracking via compteur de braces

5. **deprecated_function** (minor, 0.92 confidence)
   - Détecte: `ioutil.ReadFile`, `ioutil.WriteFile`, `ioutil.WriteDir`, etc.
   - Suggest replacements (ex: `os.ReadFile`, `os.WriteFile`)

6. **missing_error_handling** (minor, ~0.9 confidence)
   - Détecte: Appels de fonctions retournant `error` sans vérification `if err != nil`
   - Skip comments, defer, blanks (`_ =`)

### Analyse LLM
- **Status** : ✅ Implémenté (NVIDIA API)
- **Fonction** : `LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error)`
- **API** : HTTP POST vers `https://integrate.api.nvidia.com/v1/chat/completions`
- **Auth** : `NVIDIA_API_KEY` env var (Bearer token)
- **Modèle défaut** : `google/gemma-3n-e2b-it` (configurable via `cfg.Model`)
- **Usage** : Analyse contextuelle, patterns complexes, détection vulnérabilités avancée
- **Setup** : https://build.nvidia.com/google/gemma-3n-e2b-it → créer clé API

### Structure de Sortie
```go
type AnalysisResult struct {
  Timestamp   time.Time // Quand l'analyse s'est exécutée
  DiffHash    string    // Hash du diff analysé
  FileCount   int       // Nombre de fichiers modifiés
  TotalLines  int       // Lignes totales du diff
  Issues      []Issue   // Liste des problèmes détectés
  Summary     Summary   // Résumé (counts, quality score, avg confidence)
  Duration    float64   // Temps d'exécution en ms
}

type Issue struct {
  ID          string    // "secret-api_key-42" ou "sql-inj-10"
  Type        string    // "hardcoded_secrets", "sql_injection", etc.
  Severity    string    // "critical", "major", "minor"
  Location    Location  // File + StartLine + EndLine
  Message     string    // Descriptif du problème
  Suggestion  string    // How to fix
  Confidence  float64   // 0.0-1.0 (0.85-0.99 typique)
  Source      string    // "local_analyzer" ou "llm_analyzer"
}
```

## 3. Contraintes Techniques

| Aspect | Valeur |
|--------|--------|
| Langage | Go 1.21+ |
| Entrée | Unified diff (format `git diff`) |
| Parsing | Regex `@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@` pour hunks |
| Concurrence | Goroutines + WaitGroup + buffered channels |
| Package Go | Tous les fichiers d'un package **doivent** être dans le même répertoire |
| Dépendances | `anthropic-sdk-go`, `spf13/cobra` |
| Config | YAML (`.code-review-agent.yml`) + env vars |
| Cache | File-based (répertoire + TTL) |
| Formats de sortie | JSON, Markdown, CLI |

## 4. Structure Fichier/Répertoire (Actualisée)

```
internal/analyzer/
├── local/                          # Package "local"
│   ├── analyzer.go                 # LocalAnalyze(hunks) → []Issue (orchester goroutines)
│   ├── types.go                    # AnalysisRule interface, RuleRegistry, 6 rule wrappers
│   ├── analyzer_test.go            # 12 tests (tous passent ✓)
│   └── rules/                      # Package "rules" (sous-dossier = package séparé)
│       ├── init.go                 # Exports: CheckXXX() wrappers
│       ├── hardcoded_secrets.go    # checkHardcodedSecrets()
│       ├── sql_injection.go        # checkSQLInjection()
│       ├── todo_comment.go         # checkTodoComment()
│       ├── large_function.go       # checkLargeFunction()
│       ├── deprecated_function.go  # checkDeprecatedFunction()
│       └── missing_error_handling.go # checkMissingErrorHandling()
└── llm/                            # Package "llm"
    ├── analyzer.go                 # LLMAnalyze(hunks, cfg) → ([]Issue, error) + tests
    ├── prompt.go                   # BuildPrompt(hunks) + ParseLLMResponse(raw)
    └── analyzer_test.go            # 4 tests (tous passent ✓)

internal/storage/
├── sqlite.go                       # SQLite persistence layer (Phase 2)
│   ├── Store struct + methods
│   ├── NewStore(dbPath) → *Store
│   ├── Migrate() → tables (repositories, analyses, issues)
│   ├── UpsertRepository(name, path) → int64
│   ├── CreateAnalysis(repoID, result) → int64
│   ├── UpdateAnalysisResult(diffHash, result)
│   └── ListAnalysesForRepo(repoID) → []AnalysisResult
└── sqlite_test.go                  # 6 tests unitaires (tous passent ✓)
```

### Fichiers Clés Externes
| Fichier | Rôle |
|---------|------|
| `cmd/main.go` | Entry point CLI (v0.1, stub) |
| `internal/models/types.go` | DiffHunk, Issue, Location, Summary structs |
| `internal/parser/diff.go` | ParseDiff(string) → []DiffHunk |
| `internal/config/config.go` | Config structs + LoadConfig() ✅ Implémenté |
| `internal/formatter/` | JSON/Markdown/CLI output formatters ✅ Implémentés |
| `internal/cache/filedb.go` | File-based cache manager ✅ Implémenté |
| `internal/storage/sqlite.go` | SQLite persistence (Phase 2) ✅ Implémenté |

### Fichiers LLM Implémentés

**`internal/analyzer/llm/analyzer.go`**
- `LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error)`
- Effectue requête HTTP POST à `https://integrate.api.nvidia.com/v1/chat/completions`
- Lit API key depuis variable d'environnement `NVIDIA_API_KEY`
- Headers : `Content-Type: application/json`, `Authorization: Bearer {key}`
- Messages : system prompt + user content (format OpenAI-compatible)
- Modèle : `google/gemma-3n-e2b-it` (optimisé pour la détection de vulnérabilités)
- Retourne `[]models.Issue` avec `Source="llm_analyzer"`
- Gère les erreurs (API call, parsing) avec messages descriptifs

**`internal/analyzer/llm/prompt.go`**
- `BuildPrompt(hunks []models.DiffHunk) string` — Formate hunks en diff lisible (fichier, numéros lignes, contenu)
- `ParseLLMResponse(raw string) []models.Issue` — Parse JSON retourné par API Claude en `[]models.Issue`
  - Champs attendus : `type`, `severity`, `file`, `start_line`, `message`, `suggestion`, `confidence`
  - Gère gracieusement JSON invalide ou tableau vide (retourne `[]Issue{}`

## 5. Flux de Données

```
Git Diff (unified format)
    ↓
ParseDiff() → []DiffHunk {File, StartLine, AddedLines, RemovedLines, Context}
    ↓
LocalAnalyze(hunks) :
  ├─ Concurrent: Rule1.Check(hunks) → [Issue1, Issue2]
  ├─ Concurrent: Rule2.Check(hunks) → [Issue3]
  └─ Concurrent: Rule6.Check(hunks) → [Issue4, Issue5]
    ↓
Aggregation → []Issue (local issues)
    ↓
[OPTIONAL] LLMAnalyze(hunks, cfg) → []Issue (si cfg.AIEnabled)
    ↓
Merge Local + LLM → []Issue (all issues)
    ↓
AnalysisResult {Issues[], Summary{Counts, Quality, Confidence}}
    ↓
Formatter (JSON/Markdown/CLI) → Output
```

## 6. État du Projet

### ✅ Complété
- **Structure de dossiers** : Réorganisation en `local/rules/` + `llm/` (via package separation)
- **Analyse locale** : 6 rules implémentées, testées (12 tests passent)
- **Diff Parser** : Fonctionne, teste les hunks correctement
- **Models** : Issue, Location, AnalysisResult structures
- **Patterns de sécurité** : Secrets, SQL injection détectés
- **Concurrence** : LocalAnalyze orchestre les règles via goroutines
- **LLM Analyzer** : `llm/analyzer.go` + `llm/prompt.go` implémentés avec appels HTTP à NVIDIA API (4 tests passent + skips quand NVIDIA_API_KEY non défini)
- **Result Aggregator** : `internal/aggregator/aggregator.go` implémenté (merge, deduplicate, sort, summary) (13 tests passent)
- **Config Loading** : `LoadConfig()` implémenté avec support YAML + env vars (hiérarchie: CLI → env → YAML → defaults)
- **Output Formatters** : JSON, CLI (avec ANSI colors), Markdown implémentés (3 tests passent)
- **Cache Layer** : File-based cache avec TTL support implémenté (5 tests passent)
- **Main CLI** : `cmd/main.go` + `cmd/analyze.go` + `cmd/batch.go` + `cmd/cache.go` + `cmd/config.go` - Cobra CLI complète avec tous les subcommands
- **Integration Tests** : `tests/integration_test.go` avec 6 tests end-to-end (ParseDiff, LocalAnalyze, Formatter outputs)
- **Security Fixes** : 7 vulnérabilités SonarQube corrigées (SHA-256, permissions 0600, path traversal, MkdirAll idempotency)
- **LLM Provider Switch** : Migration de Mistral AI vers NVIDIA API (google/gemma-3n-e2b-it) pour meilleure détection vulnérabilités
- **Docker** : Dockerfile multi-stage, docker-compose.yml, .dockerignore, test scripts, Makefile.docker (✅ Complet)
- **Deployment Docs** : DOCKER.md + DEPLOYMENT.md avec guides complets, scripts de test, Kubernetes templates
- **Environment Setup** : .env.example créé, scripts/docker-test.sh pour validation
- **README Documentation** : Mise à jour complète avec exemples CLI + Docker, statut du projet, flux de données, démonstration d'analyse (test_vuln.diff → 8 issues)
- **Docker Validation** : docker-compose.yml sans warning de version, batch processing ✅, named volumes ✅, permissions ✅
- **GitHub Actions CI/CD** : `.github/workflows/security-check.yml` avec analyse automatique PR/push, commentaires sur PR, blocage merge si critiques, `GITHUB_ACTIONS_SETUP.md` documentation complète
- **Phase 2 : Storage SQLite** : `internal/storage/sqlite.go` avec tables (repositories, analyses, issues), API CRUD complète (NewStore, Migrate, UpsertRepository, CreateAnalysis, UpdateAnalysisResult, ListAnalysesForRepo), WAL mode + busy_timeout, 6 tests unitaires passants (100% ✅)

### 🔄 En Cours / Stubs
(Aucun)

### 📋 À Faire
(Aucun — **Projet complété** ✅)

## 7. Commandes de Test Complètes

### Tests Unitaires
```bash
# Tous les tests
go test ./... -v

# Tests spécifiques par module
go test ./internal/analyzer/local -v
go test ./internal/analyzer/llm -v
go test ./internal/cache -v
go test ./internal/aggregator -v
go test ./internal/formatter -v
go test ./internal/parser -v
go test ./tests -v  # Tests d'intégration
```

### Tests Manuels - Analyse Locale (sans LLM)
```bash
# Format CLI
./code-review-agent analyze --file=test_vuln.diff --format=cli

# Format JSON
./code-review-agent analyze --file=test_vuln.diff --format=json

# Format Markdown
./code-review-agent analyze --file=test_vuln.diff --format=markdown
```

### Tests Manuels - Avec LLM NVIDIA
```bash
# Charger les variables d'environnement
set -a && source .env && set +a

# Analyse complète (local + LLM NVIDIA)
./code-review-agent analyze --file=test_vuln.diff --llm --format=cli --verbose

# Format JSON avec LLM
./code-review-agent analyze --file=test_vuln.diff --llm --format=json --verbose
```

### Tests Batch (plusieurs diffs)
```bash
mkdir test_diffs
cp test_vuln.diff test_diffs/

./code-review-agent batch --dir=test_diffs --output=results.md --verbose
```

### Tests Cache
```bash
# Analyser un fichier (cache dans ~/.cache/code-review-agent)
./code-review-agent analyze --file=test_vuln.diff --format=json

# Voir le cache
ls -la ~/.cache/code-review-agent/

# Nettoyer le cache
./code-review-agent cache clear
```

### Build & Vérification
```bash
# Build
go build -o code-review-agent ./cmd

# Tests avec code coverage
go test ./... -cover

# Code quality
go fmt ./...
go vet ./...
```

### Quick Test (tous les tests en 1 commande)
```bash
echo "=== Tests ===" && go test ./... && \
echo "=== Build ===" && go build -o code-review-agent ./cmd && \
echo "=== CLI Test ===" && \
set -a && source .env && set +a && \
./code-review-agent analyze --file=test_vuln.diff --llm --format=cli --verbose && \
echo "✅ Tous les tests réussis!"
```

## 8. Règles Anti-Gaspi (IMPORTANT)

**🚫 SI TÂCHE TERMINÉE** : Réponds par `FAIT.` + max 1 phrase descriptive. Zéro narration de processus.

Exemples :
- ✅ "FAIT. Règle deprecated_function ajoutée avec 92% confidence."
- ✅ "FAIT. Tests passent (12/12)."
- ❌ "J'ai créé un nouveau fichier, puis j'ai modifié le parseur, ensuite j'ai lancé les tests..."

**🎯 Avant chaque action** : Vérifier dans ce fichier si elle est déjà DONE.

**📝 Mises à jour** : Après chaque changement, updater immédiatement cette section "État du Projet" pour refléter la réalité.
