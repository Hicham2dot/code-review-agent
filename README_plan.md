# PLAN TECHNIQUE : Agent IA Code Review
## Version SOLO & STANDALONE (4 semaines)

**Contexte** : 1 développeur, 1 mois, pas d'infra existante (pas de BD, pas de CI/CD)  
**Objectif** : Livrer un **outil fonctionnel et autonome**, facilement testable et intégrable

---

## 1. ANALYSE DU PROBLÈME (Revisitée)

### 1.1 Objectif Principal Simplifié
Créer un **CLI tool + API simple** qui :
- ✅ Analyse un diff/Pull Request
- ✅ Retourne des critiques intelligentes via IA
- ✅ Facile à invoquer localement ou en CI/CD
- ✅ **Pas de dépendances** : fonctionne standalone

### 1.2 Use Cases Réalistes
```
1. Local Usage (Dev):
   $ code-review-agent analyze --file=my_changes.diff
   → JSON output → affichage formaté

2. Git Hook (Pre-commit):
   $ git hook → appelle agent → bloque si CRITICAL
   
3. Futur CI/CD:
   $ docker run code-review-agent analyze --pr-id=123
   → Parse diff via API Git provider
   → Retourne rapport

4. Batch Mode:
   $ code-review-agent batch --repo=/path/to/repo
   → Analyse tous les fichiers modified
```

### 1.3 Utilisateurs Cibles (Solo)
| Rôle | Besoins |
|------|---------|
| **Toi (Dev)** | Tool rapide & précis, facile à améliorer |
| **Futurs reviewers** | Output clair, suggetsions actionnables |
| **Entreprise** | Intégrable facilement dans leurs workflows |

### 1.4 Contraintes (Adapter à Solo)

#### Techniques
- ✅ Go (efficace, binary unique)
- ✅ **Pas de DB** → Fichiers JSON locaux (cache, historique)
- ✅ **Pas de CI/CD existant** → Livrer CLI + Docker optional
- ✅ Minimal dependencies (Go stdlib mostly)
- ✅ Fonctionnement **offline** (local LLM) + **online** (API Claude/GPT)

#### Métier
- Agent = **avis**, jamais un blocage obligatoire
- Sortie JSON + Markdown (flexible pour intégration)
- Pas de webhooks complexes (utiliser Git APIs simples)

#### Ressources
- 1 mois
- ~30-35 heures/semaine de dev
- Accès API Claude/GPT (ou LLaMA local)

---

## 2. ARCHITECTURE SIMPLIFIÉE

### 2.1 Vue d'Ensemble (Ultra-Simple)
```
┌──────────────────────────────────────────────┐
│        Code Review Agent CLI Tool            │
│         (Single Binary: Go)                  │
└──────────────────┬───────────────────────────┘
                   │
        ┌──────────┼──────────┬──────────┐
        ▼          ▼          ▼          ▼
   ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐
   │ Diff   │ │  Code  │ │  LLM   │ │ Output │
   │ Parser │ │Analyzer│ │ Client │ │ Format │
   └────────┘ └────────┘ └────────┘ └────────┘
        │          │          │          │
        └──────────┴──────────┴──────────┘
                   │
        ┌──────────▼──────────┐
        │  Cache (JSON Files) │
        │   (optional)        │
        └─────────────────────┘
```

### 2.2 Composants Clés (Minimaux)

#### 1️⃣ **Diff Parser**
- Lire fichier `.diff` ou `STDIN`
- Extraire hunks (sections modifiées)
- Identifier languages
- Comptage lignes ajoutées/supprimées
- **Output** : Struct Go structuré

#### 2️⃣ **Code Analyzer (Local)**
- Détection patterns basiques (sans IA):
  - SQL injection risks
  - Hardcoded credentials
  - Missing error handling
  - TODO/FIXME comments
  - Large functions (> 50 lines)
- **Output** : Liste d'issues structurées

#### 3️⃣ **LLM Client**
- Claude API (recommandé: bon coût/qualité)
- GPT-4 (+ cher, mais complet)
- Ollama local (LLaMA 2, gratuit, private)
- **Fallback** : Si offline, retour analyses locales seules

#### 4️⃣ **Result Aggregator**
- Fusionner analyses locales + IA
- Dédupliquer
- Prioriser par severity
- Format structuré (JSON)

#### 5️⃣ **Output Formatter**
- JSON (pour parsing)
- Markdown (pour humans)
- Colored CLI output
- Exit codes (0 = pas d'issues, 1 = warnings, 2 = critical)

#### 6️⃣ **Cache Layer** (Optional)
- Stocker analyses précédentes (JSON files)
- Éviter re-analyser même diff
- Lightweight (pas de DB)

---

## 3. STRUCTURE PROJET GO

```
code-review-agent/
├── cmd/
│   └── main.go              # Entry point CLI
├── internal/
│   ├── parser/
│   │   ├── diff.go          # Diff parsing
│   │   └── diff_test.go
│   ├── analyzer/
│   │   ├── local.go         # Static pattern matching
│   │   ├── llm.go           # LLM client (Claude/GPT/Ollama)
│   │   └── analyzer_test.go
│   ├── models/
│   │   └── types.go         # Structs (Issue, Result, etc.)
│   ├── formatter/
│   │   ├── json.go
│   │   ├── markdown.go
│   │   └── cli.go
│   ├── cache/
│   │   └── filedb.go        # JSON file-based cache
│   └── config/
│       └── config.go        # Config management (flags + env vars)
├── scripts/
│   ├── test.sh
│   ├── build.sh
│   └── docker_build.sh
├── tests/
│   ├── fixtures/            # Sample diffs for testing
│   │   ├── security_issue.diff
│   │   ├── performance.diff
│   │   └── clean.diff
│   └── integration_test.go
├── examples/
│   ├── sample.diff
│   └── usage.sh
├── Dockerfile
├── docker-compose.yml       # Optional: local Ollama
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── ARCHITECTURE.md
```

---

## 4. MODÈLE DE DONNÉES

### 4.1 Structs Principaux (Go)

```go
// models/types.go

type Issue struct {
    ID          string   `json:"id"`           // unique ID (hash)
    Type        string   `json:"type"`         // "Security", "Performance", etc.
    Severity    string   `json:"severity"`     // "CRITICAL", "MAJOR", "MINOR", "INFO"
    Location    Location `json:"location"`
    Message     string   `json:"message"`
    Suggestion  string   `json:"suggestion"`
    Confidence  float64  `json:"confidence"`   // 0.0-1.0
    Source      string   `json:"source"`       // "local", "llm", "sonarqube"
}

type Location struct {
    File      string `json:"file"`
    StartLine int    `json:"start_line"`
    EndLine   int    `json:"end_line"`
}

type AnalysisResult struct {
    Timestamp   time.Time `json:"timestamp"`
    DiffHash    string    `json:"diff_hash"`
    FileCount   int       `json:"file_count"`
    TotalLines  int       `json:"total_lines"`
    Issues      []Issue   `json:"issues"`
    Summary     Summary   `json:"summary"`
    Duration    float64   `json:"duration_ms"`
}

type Summary struct {
    CriticalCount int     `json:"critical_count"`
    MajorCount    int     `json:"major_count"`
    MinorCount    int     `json:"minor_count"`
    TotalIssues   int     `json:"total_issues"`
    Quality       string  `json:"quality"` // "EXCELLENT", "GOOD", "NEEDS_REVIEW", "CRITICAL"
    Confidence    float64 `json:"avg_confidence"`
}

type DiffHunk struct {
    File        string
    StartLine   int
    EndLine     int
    RemovedLines []string
    AddedLines  []string
    Context     string // surrounding code for context
}
```

### 4.2 Cache Format (JSON)

```json
{
  ".cache/analyses/diff_hash_abc123.json": {
    "timestamp": "2026-04-19T10:30:00Z",
    "diff_hash": "abc123",
    "result": {
      "issues": [...],
      "summary": {...}
    }
  }
}
```

### 4.3 Config Format (YAML + Env Vars)

```yaml
# .code-review-agent.yml (optional)
llm:
  provider: "claude"  # or "openai", "ollama"
  model: "claude-opus-4-20250514"
  max_tokens: 2000
  temperature: 0.2

cache:
  enabled: true
  dir: ".cache/analyses"
  ttl_hours: 24

analysis:
  local_checks: true
  ai_enabled: true
  confidence_threshold: 0.6

output:
  format: "markdown"  # or "json", "cli"
  colors: true
```

---

## 5. ROADMAP 4 SEMAINES (SOLO)

### Semaine 1 : Foundation (7-9 heures)

#### Jour 1-2 (4h) : Setup & Scaffolding
- [ ] Repo Go + module setup
- [ ] Structure dossiers
- [ ] Makefile basique
- [ ] GitHub Actions workflow simple (build + test)

**Deliverable** : Binary compilé, repo clean

#### Jour 3-4 (3h) : Diff Parser MVP
- [ ] Parser diffs (format unified diff)
- [ ] Extract hunks
- [ ] Identify languages
- [ ] Unit tests

**Test** :
```bash
$ cat sample.diff | code-review-agent parse
# Output: JSON structuré des hunks
```

#### Jour 5 (2h) : Config + CLI Flags
- [ ] Parse flags (--file, --format, --provider)
- [ ] Env var support (API_KEY, etc.)
- [ ] Help text
- [ ] Default values

**Deliverable** : CLI fonctionnelle basique

---

### Semaine 2 : Local Analysis (7-9 heures)

#### Jour 6-7 (4h) : Local Analyzer (Pattern Matching)
Implémente détections simples (regex + AST basique):
- [ ] SQL injection patterns (string interpolation en SQL)
- [ ] Hardcoded secrets (regex: `password=`, `api_key=`)
- [ ] Missing error handling (`_ = func()`)
- [ ] Large functions (> 50 lines)
- [ ] TODO/FIXME comments
- [ ] Deprecated functions

**Pattern Examples** :
```go
// SQL Injection risk
query := "SELECT * FROM users WHERE id = " + id  // ⚠️ CRITICAL

// Hardcoded secret
const apiKey = "sk-1234567890abcdef"  // ⚠️ CRITICAL

// Missing error handling
_ = os.Remove(file)  // ⚠️ MAJOR

// Large function
func processData() {  // 120 lines  ⚠️ MINOR
```

#### Jour 8-9 (3h) : Unit Tests + Fixtures
- [ ] Test fixtures (sample diffs)
- [ ] Test cases per pattern
- [ ] Coverage > 70%

**Deliverable** :
```bash
$ code-review-agent analyze --file=security.diff --format=json
# Output: JSON avec issues détectées localement
```

#### Jour 10 (2h) : Output Formatters
- [ ] JSON formatter
- [ ] Markdown formatter
- [ ] CLI pretty-print

---

### Semaine 3 : LLM Integration (10-12 heures)

#### Jour 11-12 (4h) : Claude API Client
- [ ] HTTP client (net/http)
- [ ] Request/response structs
- [ ] Error handling (rate limits, network errors)
- [ ] Streaming support (optional, pour perf)
- [ ] Retry logic (exponential backoff)

**Code Skeleton** :
```go
type LLMClient interface {
    AnalyzeCode(ctx context.Context, diff string) ([]Issue, error)
}

type ClaudeClient struct {
    apiKey    string
    modelID   string
    baseURL   string
}

func (c *ClaudeClient) AnalyzeCode(ctx context.Context, diff string) ([]Issue, error) {
    // Call Claude API
    // Parse response
    // Return []Issue
}
```

#### Jour 13-14 (4h) : Prompt Optimization
Tester / ajuster prompts pour meilleure qualité:
- [ ] Prompt basique (générique)
- [ ] Prompt strict (security-focused)
- [ ] Prompt performance (backend optimization)
- [ ] Few-shot examples (si budget Claude permet)

**Prompt Template** :
```
You are an expert code reviewer.

Analyze this diff and report ONLY HIGH-CONFIDENCE issues:

DIFF:
{DIFF}

RESPOND ONLY as JSON:
{
  "issues": [
    {
      "type": "Security|Performance|Style|Logic",
      "severity": "CRITICAL|MAJOR|MINOR",
      "location": "file.go:line:col",
      "message": "...",
      "suggestion": "...",
      "confidence": 0.95
    }
  ]
}
```

#### Jour 15 (4h) : Error Handling & Offline Fallback
- [ ] API error handling (rate limits, auth errors)
- [ ] Network errors → fallback to local analysis
- [ ] Timeout handling (max 20s per request)
- [ ] Graceful degradation

**Deliverable** :
```bash
$ code-review-agent analyze --file=complex.diff --provider=claude
# → Calls Claude API, formats output, returns issues
```

---

### Semaine 4 : Polish & Testing (8-10 heures)

#### Jour 16-17 (4h) : Integration Testing
- [ ] End-to-end tests (diff → local + Claude → output)
- [ ] Test with real code samples
- [ ] Benchmark (performance)
- [ ] Memory profiling (pprof)

**Test Scenarios** :
```go
// Test 1: Simple security issue
// Test 2: False positive handling
// Test 3: Large diff (> 1000 lines)
// Test 4: Offline mode
// Test 5: API error handling
```

#### Jour 18 (2h) : Docker & Deployment
- [ ] Dockerfile (minimal, ~50MB)
- [ ] docker-compose.yml (optional: Ollama)
- [ ] Build scripts
- [ ] Documentation

**Dockerfile** :
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /usr/local/bin/code-review-agent ./cmd/main.go

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/code-review-agent /
ENTRYPOINT ["/code-review-agent"]
```

#### Jour 19 (2h) : Documentation
- [ ] README.md (installation, usage, examples)
- [ ] ARCHITECTURE.md (ce qu'on fait ici)
- [ ] DEVELOPMENT.md (comment contribuer)
- [ ] Examples/ (sample inputs + outputs)

#### Jour 20 (2h) : Polish & Bugfixes
- [ ] Fix bugs trouvés en testing
- [ ] Code cleanup
- [ ] Final optimizations
- [ ] Version bump

**Deliverable** :
```bash
# Final binary should work like:
$ code-review-agent analyze --file=diff.patch --format=markdown
# Outputs beautifully formatted review

$ code-review-agent analyze --file=diff.patch --format=json
# Outputs JSON parseable

$ code-review-agent cache --clear
# Manage local cache
```

---

## Timeline Visuelle

```
Semaine 1: Foundation        │ Diff Parser + CLI
Semaine 2: Local Analysis    │ Pattern Matching + Formatters
Semaine 3: LLM Integration   │ Claude API + Prompts
Semaine 4: Polish & Deploy   │ Tests + Docs + Docker

Day    1  2  3  4  5  6  7  8  9 10 11 12 13 14 15 16 17 18 19 20
                                         │  │  │  │  │  │  │  │  │
                                         └──LLM + Optimization──┘
```

---

## 6. TECHNOLOGY STACK (MINIMAL)

### Core
| Component | Technology | Justification |
|-----------|------------|---------------|
| **Language** | Go 1.21+ | Binary unique, fast, stdlib riche |
| **HTTP** | `net/http` | Stdlib, pas de dependency |
| **CLI** | `flag` (stdlib) ou `cobra` | Simple ou advanced |
| **JSON** | `encoding/json` | Stdlib |
| **Diff parsing** | `github.com/go-diff/diff` | ~5KB, spécialisé |
| **Cache** | JSON files | Zéro DB |

### AI Integration
| Component | Option 1 | Option 2 | Option 3 |
|-----------|----------|----------|----------|
| **LLM** | Claude API | GPT-4 | Ollama (LLaMA 2) |
| **SDK** | anthropic-sdk-go | openai-go | ollama/ollama (HTTP) |
| **Cost** | $0.003/1K tokens | $0.015/1K tokens | $0 |
| **Latency** | 1-2s | 1-2s | 0.3-1s (local) |
| **Setup** | API key only | API key only | Docker image |

### Optional
```go
// If you want prettier CLI:
github.com/fatih/color           // Colored output
github.com/jedib0t/go-pretty/v6  // Tables

// If you want structured logging:
go.uber.org/zap                  // Logging

// If you want testing helpers:
github.com/stretchr/testify      // Assert library
```

### Go Modules (Minimal)
```
go.mod:
module github.com/yourusername/code-review-agent

go 1.21

require (
    github.com/go-diff/diff v1.4.0
    // Optional:
    // github.com/anthropics/anthropic-sdk-go v0.x
)
```

**Total Dependencies** : ~2-3 external packages (très light)

---

## 7. RISQUES & MITIGATIONS (SOLO VERSION)

### 7.1 Risques Techniques

#### ⚠️ Risque: LLM API Downtime
**Impact**: Tool devient inutile si Claude/OpenAI down

**Mitigation**:
1. ✅ Local analyzer fallback (continue sans IA)
2. ✅ Cache recent results (evite re-calls)
3. ✅ Support Ollama local (offline)
4. ✅ User notification ("API unavailable, using local analysis only")

#### ⚠️ Risque: Token costs spiral
**Impact**: Budget API dépassé si PRs énormes

**Mitigation**:
1. ✅ Truncate large diffs (max 4000 tokens)
2. ✅ Sample mode (analyze only first N files)
3. ✅ Cache aggressive (don't re-analyze)
4. ✅ Offline mode by default (use `--with-ai` flag)

#### ⚠️ Risque: Faux positifs = distrust
**Impact**: Si agent crie au loup trop souvent, personne l'écoute

**Mitigation**:
1. ✅ Confidence threshold (only report > 0.7 confidence)
2. ✅ Prompt strict ("report ONLY certain issues")
3. ✅ Local checks only (no flaky IA for obvious things)
4. ✅ User feedback loop (easy to report "false alarm")

#### ⚠️ Risque: Mauvaise quality code
**Impact**: Tool generates bogus reviews

**Mitigation**:
1. ✅ Test fixtures (test against known issues)
2. ✅ Manual review of samples
3. ✅ Prompt tuning (iterate)
4. ✅ Version control for prompts (track what worked)

---

### 7.2 Risques Métier

#### ⚠️ Risque: Ne détecte pas vrais bugs
**Impact**: False sense of security

**Mitigation**:
1. ✅ **Tool aide, pas remplace** (marqué clairement)
2. ✅ Combine local + LLM (catch different issue types)
3. ✅ Highlight unknowns ("confidence: 0.3" = not sure)

#### ⚠️ Risque: Integration friction
**Impact**: Nice tool but nobody uses it

**Mitigation**:
1. ✅ CLI ultra-simple (one command)
2. ✅ Multiple output formats (CLI, JSON, Markdown)
3. ✅ Git hooks (auto-run pre-commit)
4. ✅ Good docs + examples

---

### 7.3 Risques de Sécurité

#### ⚠️ Risque: Code sent to LLM (privacy)
**Impact**: Proprietary code leaks

**Mitigation**:
1. ✅ **Local mode by default** (use `--provider=local`)
2. ✅ Warn user if sending to API
3. ✅ Support Ollama (fully local, private)
4. ✅ Terms of service check (Claude doesn't store code)
5. ✅ Option to redact secrets before sending

#### ⚠️ Risque: API key exposed
**Impact**: Token stolen, costs spirale

**Mitigation**:
1. ✅ Use env vars (`CLAUDE_API_KEY`)
2. ✅ Never log API keys
3. ✅ Validate on startup (early fail)
4. ✅ .gitignore template in repo

---

### Risk Matrix (Solo)
| Risque | Severity | Probability | Solution |
|--------|----------|-------------|----------|
| API downtime | High | Medium | Offline fallback |
| Token costs | Medium | High | Truncation + cache |
| Faux positifs | High | High | Prompt tuning |
| Privacy leak | Critical | Low | Local mode default |
| Not adopted | Medium | Medium | Good UX + docs |

---

## 8. QUICK START GUIDE (After Completion)

### Pour toi (Dev)
```bash
# Clone + Setup
git clone <your-repo>
cd code-review-agent
make build

# Local testing
./code-review-agent analyze --file=sample.diff --format=markdown

# With AI
export CLAUDE_API_KEY=sk-...
./code-review-agent analyze --file=diff.patch --provider=claude

# Offline mode
./code-review-agent analyze --file=diff.patch --provider=local
```

### Pour quelqu'un d'autre
```bash
# Docker
docker build -t code-review-agent .
docker run -e CLAUDE_API_KEY=sk-... \
  code-review-agent analyze --file=/tmp/diff.patch

# Or: installé dans PATH
code-review-agent analyze --help
```

### Integration examples

#### Git Pre-commit Hook
```bash
#!/bin/bash
# .git/hooks/pre-commit
git diff --cached > /tmp/staged.diff
if code-review-agent analyze --file=/tmp/staged.diff --severity=CRITICAL | grep -q CRITICAL; then
    echo "❌ Critical issues found. Fix before commit."
    exit 1
fi
echo "✅ Code review passed"
exit 0
```

#### GitHub Actions (Future)
```yaml
name: Code Review AI
on: [pull_request]
jobs:
  review:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - run: docker run code-review-agent analyze --pr-url=${{ github.event.pull_request.url }}
```

---

## 9. SUCCESS METRICS (4 weeks)

### Functional
- ✅ CLI tool compiles & runs
- ✅ Can parse real diffs
- ✅ Detects obvious issues (SQL injection, hardcoded secrets)
- ✅ LLM integration works (Claude or offline)
- ✅ Output in JSON + Markdown
- ✅ Local cache functioning

### Quality
- ✅ Test coverage > 60%
- ✅ No crashes on edge cases
- ✅ False positive rate < 10% (on test set)
- ✅ Confidence scoring accurate

### Documentation
- ✅ README with examples
- ✅ ARCHITECTURE.md (for future contributors)
- ✅ Sample diffs + expected outputs
- ✅ Troubleshooting section

---

## 10. NEXT STEPS (Immediately)

### This Week
- [ ] Create GitHub repo
- [ ] Setup Go module
- [ ] Create basic project structure
- [ ] Write Makefile
- [ ] Verify Claude API access

### Week 1 Priority
1. **Diff parser** = core of everything
2. **Basic CLI** = user-facing
3. **Unit tests** = confidence

### Git Commit Discipline
```
Commit messages:
✨ feat: diff parser MVP
🐛 fix: handle binary diffs
🧪 test: add parser fixtures
📖 docs: usage examples
♻️ refactor: consolidate error handling
```

---

## 11. ARCHITECTURE DECISIONS (Justified)

### Why Go?
- ✅ Single binary (no runtime dependencies)
- ✅ Fast compilation
- ✅ Goroutines for potential parallelization
- ✅ Excellent stdlib
- ❌ But: steep learning curve if new

### Why No Database?
- ✅ Simpler (no setup, no migrations)
- ✅ Portable (works anywhere)
- ✅ Fast (JSON cache sufficient for solo)
- ❌ But: not scalable to millions of analyses (ok for now)

### Why Offline-First?
- ✅ Privacy (default: no external API calls)
- ✅ Fast (local patterns instant)
- ✅ No API key management issues
- ✅ Works offline
- ❌ But: LLM analyses much more powerful

### Why Claude over GPT?
- ✅ Better code understanding (trained on code)
- ✅ Lower cost ($0.003 vs $0.015 per 1K tokens)
- ✅ More reliable prompt following
- ✅ 200K context window (large diffs)
- ❌ But: slower than GPT-4 (1-2s vs 0.5s)

**Alternative**: If budget/cost important, use local Ollama (free, private)

---

## 12. EXAMPLE: Day 1 ACTUAL CODE

To get started, here's what Day 1 looks like:

### `go.mod`
```go
module github.com/yourusername/code-review-agent

go 1.21
```

### `cmd/main.go`
```go
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)
	fileFlag := analyzeCmd.String("file", "", "Path to diff file")
	formatFlag := analyzeCmd.String("format", "json", "Output format: json, markdown, cli")
	providerFlag := analyzeCmd.String("provider", "local", "AI provider: local, claude, openai")

	if len(os.Args) < 2 {
		fmt.Println("Usage: code-review-agent analyze --file=<path> [--format=json|markdown|cli] [--provider=local|claude]")
		os.Exit(1)
	}

	analyzeCmd.Parse(os.Args[2:])

	if *fileFlag == "" {
		fmt.Println("Error: --file is required")
		os.Exit(1)
	}

	fmt.Printf("Analyzing diff: %s\n", *fileFlag)
	fmt.Printf("Format: %s, Provider: %s\n", *formatFlag, *providerFlag)
	fmt.Println("✨ TODO: implement analysis logic")
}
```

### `Makefile`
```makefile
.PHONY: build test clean help

build:
	go build -o ./bin/code-review-agent ./cmd/main.go
	@echo "✅ Built successfully: ./bin/code-review-agent"

test:
	go test ./... -v -cover

clean:
	rm -rf ./bin
	go clean

help:
	@echo "Makefile targets:"
	@echo "  make build    - Compile binary"
	@echo "  make test     - Run tests"
	@echo "  make clean    - Clean artifacts"
```

### `internal/models/types.go`
```go
package models

import "time"

type Issue struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`      // Security, Performance, Style, Logic
	Severity   string  `json:"severity"`  // CRITICAL, MAJOR, MINOR, INFO
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Message    string  `json:"message"`
	Suggestion string  `json:"suggestion"`
	Confidence float64 `json:"confidence"`
	Source     string  `json:"source"`    // local, llm
}

type AnalysisResult struct {
	Timestamp   time.Time `json:"timestamp"`
	TotalIssues int       `json:"total_issues"`
	Issues      []Issue   `json:"issues"`
	Summary     Summary   `json:"summary"`
}

type Summary struct {
	CriticalCount int    `json:"critical_count"`
	MajorCount    int    `json:"major_count"`
	MinorCount    int    `json:"minor_count"`
	Quality       string `json:"quality"` // EXCELLENT, GOOD, NEEDS_REVIEW, CRITICAL
}
```

---

## 13. FINAL CHECKLIST (4 weeks)

### Week 1 ✅
- [ ] Repo setup
- [ ] Diff parser working
- [ ] CLI basic
- [ ] 5+ test cases

### Week 2 ✅
- [ ] Local analyzer (10+ patterns)
- [ ] Output formatters (JSON, Markdown, CLI)
- [ ] Unit tests (coverage > 60%)

### Week 3 ✅
- [ ] Claude API client
- [ ] Prompt optimization
- [ ] Error handling + fallback
- [ ] Integration tests

### Week 4 ✅
- [ ] Docker image
- [ ] Comprehensive tests
- [ ] Documentation
- [ ] Code cleanup
- [ ] Version 1.0.0 release

---

## 14. CONCLUSION

**Avantages de ce plan** :
✅ Réaliste pour 1 personne, 1 mois
✅ Zéro dépendances externes (pas de BD)
✅ Offline-first (privacy by default)
✅ CLI simple = facile à utiliser
✅ Evite gold-plating (scope manageable)
✅ Facilement intégrable (modular, API-friendly)

**Risques managés** :
✅ API costs → caching, truncation
✅ Privacy → local mode default
✅ Quality → testing, prompt tuning
✅ Adoption → good UX, documentation

**Bonus** :
🎁 Appris Go properly
🎁 Built real tool (resume-worthy)
🎁 Can show on GitHub (portfolio)
🎁 Foundation for future improvements

---

**C'est parti ? Questions sur le plan ? 🚀**
