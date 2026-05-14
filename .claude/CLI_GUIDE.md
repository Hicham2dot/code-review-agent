# Guide d'Implémentation CLI - Code Review Agent

## 1. Vue d'ensemble
La CLI utilise **Cobra** (spf13/cobra) avec structure hiérarchique:
- `analyze` : Analyser un diff
- `batch` : Analyser plusieurs diffs
- `cache` : Gérer le cache (clear, status)
- `config` : Gérer la configuration (init)

## 2. Implémentation Étapes

### Étape 1 : Root Command (`cmd/root.go`)
```go
package main

import ("fmt"; "os"; "github.com/spf13/cobra")

var rootCmd = &cobra.Command{
	Use:   "code-review-agent",
	Short: "Code Review Agent - Analyse statique et LLM de diffs",
	Version: "0.1.0",
}

var (configPath, format string; verbose bool)

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", ".code-review-agent.yml", "Config file path")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
	rootCmd.PersistentFlags().StringVar(&format, "format", "cli", "Output format: json, markdown, cli")
	rootCmd.AddCommand(analyzeCmd, batchCmd, cacheCmd, configCmd)
}
```

### Étape 2 : Analyze Command (`cmd/analyze.go`)
Étapes clés:
1. Charger config: `config.LoadConfig(configPath)`
2. Lire diff: `os.ReadFile()` ou `os.ReadAll(os.Stdin)`
3. Parser: `parser.ParseDiff(diffContent)` → `[]DiffHunk`
4. Analyse locale: `local.LocalAnalyze(hunks)` → `[]Issue`
5. Analyse LLM (optionnel): `llm.LLMAnalyze(hunks, cfg.LLM)` → `[]Issue`
6. Agrégation: `aggregator.Aggregate(hunks, allIssues)` → `AnalysisResult`
7. Formatage: `formatter.Format{JSON,CLI,Markdown}(result)` → string
8. Afficher ou sauvegarder résultat

Flags: `--file`, `--stdin`, `--llm`, `--output`

### Étape 3 : Batch Command (`cmd/batch.go`)
- Globber `.diff` files: `filepath.Glob(pattern)`
- Boucler sur fichiers, analyser chacun (étapes Analyze)
- Formatter résultats batch: `formatter.FormatBatchResults([]results, format)`

### Étape 4 : Cache Commands (`cmd/cache.go`)
Deux sous-commandes:
- `clear`: `cache.ClearCache(cacheDir)`
- `status`: `cache.GetCacheStats(cacheDir)` → {Count, Size, OldestEntry}

### Étape 5 : Config Commands (`cmd/config.go`)
- `init`: `config.SaveConfig(cfgPath, defaultConfig)`

### Étape 6 : Main (`cmd/main.go`)
```go
func main() { rootCmd.Execute() }
```

## 3. Patterns Go

**Gestion erreurs:**
```go
if err != nil {
	if verbose { fmt.Fprintf(os.Stderr, "Debug: %v\n", err) }
	return fmt.Errorf("user message: %w", err)
}
```

**Exit codes:** 0=succès, 1=erreur générale, 2=erreur config

## 4. Intégration Modules

| Module | Fonction | Signature |
|--------|----------|-----------|
| `parser` | ParseDiff | `ParseDiff(string) ([]DiffHunk, error)` |
| `local` | LocalAnalyze | `LocalAnalyze([]DiffHunk) ([]Issue, error)` |
| `llm` | LLMAnalyze | `LLMAnalyze([]DiffHunk, LLMConfig) ([]Issue, error)` |
| `aggregator` | Aggregate | `Aggregate([]DiffHunk, []Issue) AnalysisResult` |
| `formatter` | Format* | `Format{JSON,CLI,Markdown}(AnalysisResult) (string, error)` |
| `config` | LoadConfig | `LoadConfig(string) (Config, error)` |
| `cache` | ClearCache | `ClearCache(string) error` |

## 5. Test Basique
```bash
go build -o code-review-agent ./cmd
./code-review-agent analyze --file=changes.diff
./code-review-agent batch --dir=./diffs
./code-review-agent cache clear
./code-review-agent config init
```

---
**Voir CLI_IMPLEMENTATION_GUIDE.md dans .claude/ pour le code complet.**
