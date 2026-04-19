# Code Review Agent

An intelligent CLI tool and API for automated code review using AI.

## Status

[![SonarCloud](https://sonarcloud.io/api/project_badges/measure?project=code-review-agent&metric=alert_status)](https://sonarcloud.io/project/overview?id=code-review-agent)

| Composant | Statut |
|-----------|--------|
| Structure du projet | done |
| GitHub repository | done |
| SonarCloud integration | done |
| Diff Parser | TODO - Semaine 1 |
| Local Analyzer | TODO - Semaine 2 |
| LLM Integration (Claude) | TODO - Semaine 3 |
| Docker + Tests | TODO - Semaine 4 |

## Features

- AI-powered code analysis (Claude, GPT-4, Ollama)
- Local static analysis patterns
- Multiple output formats (JSON, Markdown, CLI)
- File-based caching
- Docker support
- Standalone binary

## Quick Start

```bash
make build
./code-review-agent analyze --file=my_changes.diff
```

## Usage

```bash
# Local analysis only
./code-review-agent analyze --file=changes.diff

# With AI analysis
./code-review-agent analyze --file=changes.diff --llm=claude

# Batch mode
./code-review-agent batch --repo=.

# Clear cache
./code-review-agent cache-clear
```

## Configuration

Set environment variables or use `.code-review-agent.yml`:

```yaml
llm:
  provider: claude
  model: claude-opus-4-20250514
  max_tokens: 2000

cache:
  enabled: true
  ttl_hours: 24
```

## Development

```bash
make test        # Run tests
make build       # Build binary
make docker-build # Build Docker image
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
