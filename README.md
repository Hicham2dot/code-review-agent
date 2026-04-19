# Code Review Agent

An intelligent CLI tool and API for automated code review using AI.

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

## License

MIT
