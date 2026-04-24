# Architecture

## Overview

```
CLI Tool (cmd/main.go)
  ├── Parser (diff.go)
  ├── Analyzer
  │   ├── Local (static patterns)
  │   └── LLM (AI analysis)
  ├── Formatter (JSON, Markdown, CLI)
  ├── Cache (file-based)
  └── Config (env vars + YAML)
```

## Modules

### Parser
Handles parsing of unified diff format and extraction of code changes.

### Analyzer
- **Local**: Static pattern matching (SQL injection, credentials, etc.)
- **LLM**: AI-powered analysis via Claude/GPT/Ollama

### Formatter
Outputs results in JSON, Markdown, or colored CLI format.

### Cache
File-based caching to avoid re-analyzing identical diffs.

### Config
Manages configuration from environment variables and YAML files.

## Data Flow

1. User provides diff file
2. Parser extracts changes
3. Local analyzer runs pattern matching
4. (Optional) LLM analyzer runs
5. Results aggregated and deduplicated
6. Output formatted and displayed
