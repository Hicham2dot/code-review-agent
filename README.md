# Code Review Agent

Outil CLI intelligent pour l'analyse automatique de code avec détection de vulnérabilités, via checks locaux et analyse LLM (NVIDIA API).

## ✅ Statut du Projet

[![SonarCloud](https://sonarcloud.io/api/project_badges/measure?project=code-review-agent&metric=alert_status)](https://sonarcloud.io/project/overview?id=code-review-agent)

| Composant | Statut |
|-----------|--------|
| Diff Parser | ✅ Complet |
| Local Analyzer (6 règles) | ✅ Complet |
| LLM Integration (NVIDIA API) | ✅ Complet |
| Docker + Tests | ✅ Complet |
| CLI Commands | ✅ Complet |
| Configuration & Cache | ✅ Complet |

## Features

- **6 Local Rules** : Secrets, SQL injection, TODO comments, large functions, deprecated APIs, missing error handling
- **LLM Analysis** : NVIDIA API (google/gemma-3n-e2b-it) pour détection contextuelles avancées
- **Multiple Formats** : JSON, Markdown, CLI (avec couleurs ANSI)
- **Batch Processing** : Analyse de répertoires entiers
- **File-based Cache** : Réutilisation des résultats (TTL configurable)
- **Docker Support** : Image multi-stage, docker-compose, tests intégrés
- **Standalone Binary** : Déploiement simple sans dépendances
- **GitHub Actions CI/CD** : Intégration automatique, bloque les merge si vulnérabilités critiques

## GitHub Actions (CI/CD)

Intégration automatique dans votre pipeline : détecte et bloque les push/PR dangereux.

```yaml
# .github/workflows/security-check.yml (déjà configuré)
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main, develop ]
```

**Fonctionnement** :
1. Chaque PR/push déclenche l'analyse
2. Commentaire automatique avec les résultats sur la PR
3. Merge **bloqué** si vulnérabilités critiques détectées
4. Status check échoue jusqu'à résolution

**Setup** : Voir [GITHUB_ACTIONS_SETUP.md](GITHUB_ACTIONS_SETUP.md)

## Quick Start

### Avec le binaire

```bash
# Build
go build -o code-review-agent ./cmd

# Analyse locale (sans LLM)
./code-review-agent analyze --file=changes.diff --format=cli

# Analyse complète (local + LLM NVIDIA)
export NVIDIA_API_KEY=nvapi-...
./code-review-agent analyze --file=changes.diff --llm --format=json --verbose
```

### Avec Docker

```bash
# Build image
docker build -t code-review-agent:latest .

# Analyse simple
docker run --rm \
  -v $(pwd)/changes.diff:/tmp/changes.diff \
  code-review-agent:latest \
  analyze --file=/tmp/changes.diff --format=cli

# Avec LLM NVIDIA
docker run --rm \
  -e NVIDIA_API_KEY=$NVIDIA_API_KEY \
  -v $(pwd)/changes.diff:/tmp/changes.diff \
  code-review-agent:latest \
  analyze --file=/tmp/changes.diff --llm --format=json

# Docker Compose (batch processing)
docker-compose up
```

## Usage CLI

### Commandes disponibles

```bash
./code-review-agent --help

Available Commands:
  analyze     Analyser un diff
  batch       Analyser plusieurs diffs
  cache       Gérer le cache
  config      Afficher la configuration
  help        Help about any command
```

### Exemples

```bash
# 1. Analyse locale (6 règles)
./code-review-agent analyze --file=test.diff --format=cli

# 2. Avec couleurs ANSI
./code-review-agent analyze --file=test.diff --format=cli --verbose

# 3. Format JSON pour intégration
./code-review-agent analyze --file=test.diff --format=json

# 4. Format Markdown pour rapports
./code-review-agent analyze --file=test.diff --format=markdown

# 5. Avec LLM (nécessite NVIDIA_API_KEY)
set -a && source .env && set +a
./code-review-agent analyze --file=test.diff --llm --format=cli

# 6. Batch mode (analyser un répertoire)
./code-review-agent batch --dir=./diffs --output=report.md --verbose

# 7. Voir le cache
./code-review-agent cache list

# 8. Nettoyer le cache
./code-review-agent cache clear
```

## Configuration

### Variables d'environnement

```bash
# NVIDIA API (obligatoire pour --llm)
export NVIDIA_API_KEY=nvapi-...

# Optionnel : modèle LLM (défaut: google/gemma-3n-e2b-it)
export REVIEW_LLM_MODEL=google/gemma-3n-e2b-it

# Optionnel : max tokens (défaut: 1024)
export REVIEW_LLM_MAX_TOKENS=2048
```

### Fichier `.code-review-agent.yml` (optionnel)

```yaml
llm:
  enabled: true
  model: google/gemma-3n-e2b-it
  max_tokens: 1024

cache:
  enabled: true
  ttl_hours: 24
```

## Docker

### Fichiers

- **Dockerfile** : Multi-stage, image Alpine optimisée (60 MB)
- **docker-compose.yml** : Batch processing avec volumes nommés
- **.dockerignore** : Exclusions pour optimiser le build

### Test

```bash
# Valider le setup Docker
./scripts/docker-test.sh

# Ou via Make
make docker-test
```

## Exemple d'Analyse

### Input (test_vuln.diff)

```diff
--- a/database.go
+++ b/database.go
@@ -1,3 +1,30 @@
 package main

+const apiKey = "sk-1234567890abcdef"
+const password = "admin123"
+const awsKey = "AKIA2BXQ7K5ABCDEF"
+
+func QueryDB(userInput string) {
+    query := "SELECT * FROM users WHERE id = " + userInput
+    db.Exec(query)
+}
+
+func SetupDatabase() {
+    // TODO: Handle error properly
+    db.Connect()
+}
```

### Output (CLI)

```
=== Code Review Analysis ===
Timestamp: 2026-05-17 14:37:09
Files: 1 | Total lines: 30 | Duration: 0.00 ms

Quality: D | Confidence: 0.96
Issues: 8 (Critical: 6, Major: 1, Minor: 1)

=== Issues ===
1. critical🔴 sql_injection [database.go:10]
   Message: Potential SQL injection vulnerability
   Suggestion: Use parameterized queries or prepared statements
   Confidence: 0.85 | Source: local_analyzer

2. critical🔴 hardcoded_secrets [database.go:13]
   Message: Hardcoded api_key detected in code
   Suggestion: Use environment variables or secret management service
   Confidence: 0.98 | Source: local_analyzer

3. critical🔴 hardcoded_secrets [database.go:14]
   Message: Hardcoded password detected in code
   Confidence: 0.98 | Source: local_analyzer

4. critical🔴 hardcoded_secrets [database.go:15]
   Message: Hardcoded aws_key detected in code
   Confidence: 0.98 | Source: local_analyzer

5. critical🔴 Security [database.go:10]
   Message: SQL Injection vulnerability detected
   Confidence: 1.00 | Source: llm_analyzer

6. critical🔴 Security [database.go:13]
   Message: Hardcoded API key and password in the code
   Confidence: 1.00 | Source: llm_analyzer

7. major🟡 Code Quality [database.go:20]
   Message: Error handling in SetupDatabase is incomplete
   Confidence: 0.80 | Source: llm_analyzer

8. minor🟢 todo_comment [database.go:20]
   Message: Found TODO comment: Handle error properly
   Confidence: 0.99 | Source: local_analyzer
```

## Development

```bash
# Tests
go test ./... -v

# Build
go build -o code-review-agent ./cmd

# Make targets
make build        # Build binary
make test         # Run tests
make docker-build # Build Docker image
make docker-test  # Test Docker deployment
```

## Project Structure

```
code-review-agent/
├── .github/workflows/sonar.yml   # SonarCloud CI
├── sonar-project.properties      # SonarCloud config
├── cmd/main.go                   # Entry point CLI
├── internal/
│   ├── models/types.go           # Data structures
│   ├── parser/diff.go            # Diff parser
│   ├── analyzer/local.go         # Static analysis
│   ├── analyzer/llm.go           # LLM client
│   ├── formatter/                # JSON/Markdown/CLI output
│   ├── cache/filedb.go           # File-based cache
│   └── config/config.go          # Configuration
├── tests/fixtures/               # Sample diffs
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## License

MIT
