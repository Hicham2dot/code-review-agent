# Memory Index - Code Review Agent

## Project Documentation

- [CLAUDE.md](CLAUDE.md) — Project specifications, architecture, 6 local security rules, LLM config (NVIDIA)
- [deployment.md](deployment.md) — Complete production deployment guide (Docker, CI/CD, releases, monitoring)

## Quick Links

- **Local Tests**: `go test ./... -v`
- **Full CLI Test**: `set -a && source .env && set +a && ./code-review-agent analyze --file=test_vuln.diff --llm --format=cli`
- **Build**: `go build -o code-review-agent ./cmd`
- **Docker**: `docker build -t code-review-agent:latest .`

## Key Files

| File | Purpose |
|------|---------|
| `.code-review-agent.yml` | Configuration (NVIDIA provider enabled) |
| `.env` | NVIDIA_API_KEY (local development) |
| `Dockerfile` | Docker image definition |
| `.github/workflows/` | CI/CD pipelines |

## Status

✅ **Production Ready**
- All tests passing (42 tests)
- NVIDIA LLM integration complete
- CLI fully functional (analyze, batch, cache, config)
- Documentation complete
- Deployment guide ready
