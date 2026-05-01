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
- **Status** : ✅ Implémenté (Mistral AI)
- **Fonction** : `LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error)`
- **API** : HTTP POST vers `https://api.mistral.ai/v1/chat/completions`
- **Auth** : `MISTRAL_API_KEY` env var (Bearer token)
- **Modèle défaut** : `mistral-tiny` (configurable via `cfg.Model`)
- **Usage** : Analyse contextuelle, patterns complexes, code quality subjectif
- **Setup** : https://console.mistral.ai/ → créer clé gratuite

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
```

### Fichiers Clés Externes
| Fichier | Rôle |
|---------|------|
| `cmd/main.go` | Entry point CLI (v0.1, stub) |
| `internal/models/types.go` | DiffHunk, Issue, Location, Summary structs |
| `internal/parser/diff.go` | ParseDiff(string) → []DiffHunk |
| `internal/config/config.go` | Config structs + LoadConfig() (TODO) |
| `internal/formatter/` | JSON/Markdown/CLI output formatters |
| `internal/cache/filedb.go` | File-based cache manager |

### Fichiers LLM Implémentés

**`internal/analyzer/llm/analyzer.go`**
- `LLMAnalyze(hunks []models.DiffHunk, cfg config.LLMConfig) ([]models.Issue, error)`
- Effectue requête HTTP POST à `https://api.mistral.ai/v1/chat/completions`
- Lit API key depuis variable d'environnement `MISTRAL_API_KEY`
- Headers : `Content-Type: application/json`, `Authorization: Bearer {key}`
- Messages : system prompt + user content (format OpenAI-compatible)
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
- **LLM Analyzer** : `llm/analyzer.go` + `llm/prompt.go` implémentés avec appels HTTP à Mistral AI (4 tests passent)

### 🔄 En Cours / Stubs
- **Main CLI** : `cmd/main.go` affiche usage seulement
- **Config loading** : `LoadConfig()` retourne vide (TODO)
- **Formatters** : JSON/Markdown/CLI existent mais pas intégrés

### 📋 À Faire
1. **CLI Integration** : Implémenter cobra commands (analyze, batch, cache-clear)
2. **Config Loading** : Lire `.code-review-agent.yml` + env vars
3. **Testing** : Intégration tests (tests/integration_test.go)
4. **Docker** : Valider build et image

## 7. Commandes Utiles

```bash
# Tests
go test ./internal/analyzer/local -v

# Build
go build -o code-review-agent ./cmd

# Exécution (quand CLI sera implémentée)
./code-review-agent analyze --file=changes.diff
./code-review-agent analyze --file=changes.diff --llm=claude
```

## 8. Règles Anti-Gaspi (IMPORTANT)

**🚫 SI TÂCHE TERMINÉE** : Réponds par `FAIT.` + max 1 phrase descriptive. Zéro narration de processus.

Exemples :
- ✅ "FAIT. Règle deprecated_function ajoutée avec 92% confidence."
- ✅ "FAIT. Tests passent (12/12)."
- ❌ "J'ai créé un nouveau fichier, puis j'ai modifié le parseur, ensuite j'ai lancé les tests..."

**🎯 Avant chaque action** : Vérifier dans ce fichier si elle est déjà DONE.

**📝 Mises à jour** : Après chaque changement, updater immédiatement cette section "État du Projet" pour refléter la réalité.
