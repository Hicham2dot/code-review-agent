# PLAN TECHNIQUE : Agent IA Code Review
## Version SOLO & STANDALONE (4 semaines)

**Contexte** : 1 développeur, 1 mois, pas d'infra existante  
**Objectif** : Livrer un **outil fonctionnel et autonome**

## Architecture

```
CLI Tool → Parser → Analyzer (Local + LLM) → Formatter → Output
                                    ↓
                               Cache (JSON)
```

## Roadmap

- **Semaine 1**: Foundation - Diff Parser + CLI
- **Semaine 2**: Local Analysis - Pattern Matching + Formatters  
- **Semaine 3**: LLM Integration - Claude API + Prompts
- **Semaine 4**: Polish & Deploy - Tests + Docs + Docker

## Stack

- Language: Go 1.21+
- LLM: Claude API / GPT-4 / Ollama (local)
- Cache: JSON files (no DB)
- Output: JSON, Markdown, CLI
