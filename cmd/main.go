package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"code-review-agent/internal/aggregator"
	"code-review-agent/internal/analyzer/llm"
	"code-review-agent/internal/analyzer/local"
	cachepkg "code-review-agent/internal/cache"
	"code-review-agent/internal/config"
	"code-review-agent/internal/formatter"
	"code-review-agent/internal/models"
	"code-review-agent/internal/parser"
)

var (
	configPath string
	format     string
	verbose    bool
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:     "code-review-agent",
	Short:   "Code Review Agent - Analyse statique et LLM de diffs",
	Long:    "Code Review Agent analyse automatiquement les modifications de code pour détecter les problèmes de qualité, sécurité et conformité.",
	Version: "0.1.0",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		cfg = config.LoadConfig()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", ".code-review-agent.yml", "Config file path")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
	rootCmd.PersistentFlags().StringVar(&format, "format", "cli", "Output format: json, markdown, cli")

	rootCmd.AddCommand(analyzeCmd, batchCmd, cacheCmd, configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ============================================================================
// ANALYZE COMMAND
// ============================================================================

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyser un diff",
	Long:  "Analyser un diff pour détecter les problèmes de code",
	RunE:  runAnalyze,
}

var (
	analyzeFile   string
	analyzeStdin  bool
	analyzeLLM    bool
	analyzeOutput string
)

func init() {
	analyzeCmd.Flags().StringVar(&analyzeFile, "file", "", "Chemin du fichier diff")
	analyzeCmd.Flags().BoolVar(&analyzeStdin, "stdin", false, "Lire depuis stdin")
	analyzeCmd.Flags().BoolVar(&analyzeLLM, "llm", false, "Activer analyse LLM")
	analyzeCmd.Flags().StringVar(&analyzeOutput, "output", "", "Fichier de sortie (défaut: stdout)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// 1. Lire le diff
	var diffContent string
	if analyzeFile != "" {
		data, err := os.ReadFile(analyzeFile)
		if err != nil {
			return fmt.Errorf("erreur lecture fichier: %w", err)
		}
		diffContent = string(data)
	} else if analyzeStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("erreur lecture stdin: %w", err)
		}
		diffContent = string(data)
	} else {
		return fmt.Errorf("spécifiez --file ou --stdin")
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Diff chargé (%d octets)\n", len(diffContent))
	}

	// 2. Parser le diff
	hunks := parser.ParseDiff(diffContent)

	if verbose {
		fmt.Fprintf(os.Stderr, "Diff parsé (%d hunks)\n", len(hunks))
	}

	// 3. Analyse locale
	localIssues := local.LocalAnalyze(hunks)

	if verbose {
		fmt.Fprintf(os.Stderr, "Analyse locale complétée (%d problèmes)\n", len(localIssues))
	}

	// 4. Analyse LLM (optionnelle)
	var llmIssues []models.Issue
	if analyzeLLM && cfg.Analysis.AIEnabled {
		var err error
		llmIssues, err = llm.LLMAnalyze(hunks, cfg.LLM)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Avertissement: analyse LLM échouée: %v\n", err)
			}
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Analyse LLM complétée (%d problèmes)\n", len(llmIssues))
		}
	}

	// 5. Calcul du hash du diff
	h := md5.Sum([]byte(diffContent))
	diffHash := fmt.Sprintf("%x", h)

	// 6. Agrégation
	result := aggregator.Aggregate(localIssues, llmIssues, hunks, diffContent)
	result.DiffHash = diffHash

	if verbose {
		fmt.Fprintf(os.Stderr, "Agrégation complétée (%d problèmes finaux)\n", len(result.Issues))
	}

	// 7. Formatage
	var output string
	switch format {
	case "json":
		output = formatter.FormatJSON(&result)
	case "markdown":
		output = formatter.FormatMarkdown(&result)
	case "cli":
		fallthrough
	default:
		output = formatter.FormatCLI(&result)
	}

	// 8. Afficher ou sauvegarder
	if analyzeOutput != "" {
		if err := os.WriteFile(analyzeOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("erreur écriture fichier: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Résultat sauvegardé dans %s\n", analyzeOutput)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}

// ============================================================================
// BATCH COMMAND
// ============================================================================

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Analyser plusieurs diffs",
	Long:  "Analyser plusieurs fichiers .diff dans un répertoire",
	RunE:  runBatch,
}

var (
	batchDir    string
	batchOutput string
)

func init() {
	batchCmd.Flags().StringVar(&batchDir, "dir", "./diffs", "Répertoire contenant les fichiers .diff")
	batchCmd.Flags().StringVar(&batchOutput, "output", "", "Fichier de sortie (défaut: stdout)")
}

func runBatch(cmd *cobra.Command, args []string) error {
	// 1. Globber les fichiers .diff
	pattern := filepath.Join(batchDir, "*.diff")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("erreur globbing: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Trouvé %d fichiers .diff\n", len(files))
	}

	// 2. Traiter chaque fichier
	var batchResults []models.AnalysisResult
	for _, filePath := range files {
		if verbose {
			fmt.Fprintf(os.Stderr, "Traitement %s...\n", filePath)
		}

		// Lire le diff
		data, err := os.ReadFile(filePath)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Erreur lecture %s: %v\n", filePath, err)
			}
			continue
		}

		diffContent := string(data)

		// Parser
		hunks := parser.ParseDiff(diffContent)

		// Analyse locale
		localIssues := local.LocalAnalyze(hunks)

		// Analyse LLM (optionnelle)
		var llmIssues []models.Issue
		if cfg.Analysis.AIEnabled {
			llmIssues, _ = llm.LLMAnalyze(hunks, cfg.LLM)
		}

		// Agrégation
		result := aggregator.Aggregate(localIssues, llmIssues, hunks, diffContent)
		h := md5.Sum([]byte(diffContent))
		result.DiffHash = fmt.Sprintf("%x", h)
		batchResults = append(batchResults, result)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Traitement batch complété (%d fichiers)\n", len(batchResults))
	}

	// 3. Formatter et afficher résultats
	output := "# Résultats Batch Analysis\n\n"
	for i, result := range batchResults {
		output += fmt.Sprintf("## Fichier %d (Hash: %s)\n", i+1, result.DiffHash)
		output += formatter.FormatMarkdown(&result)
		output += "\n---\n\n"
	}

	// 4. Afficher ou sauvegarder
	if batchOutput != "" {
		if err := os.WriteFile(batchOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("erreur écriture fichier: %w", err)
		}

		if verbose {
			fmt.Fprintf(os.Stderr, "Résultats batch sauvegardés dans %s\n", batchOutput)
		}
	} else {
		fmt.Println(output)
	}

	return nil
}

// ============================================================================
// CACHE COMMAND
// ============================================================================

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Gérer le cache",
	Long:  "Commandes pour gérer le cache (clear)",
}

var clearCacheCmd = &cobra.Command{
	Use:   "clear",
	Short: "Vider le cache",
	RunE:  runCacheClear,
}

func init() {
	cacheCmd.AddCommand(clearCacheCmd)
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	if err := cachepkg.Clear(); err != nil {
		return fmt.Errorf("erreur vidage cache: %w", err)
	}

	fmt.Println("Cache vidé avec succès")
	return nil
}

// ============================================================================
// CONFIG COMMAND
// ============================================================================

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Gérer la configuration",
	Long:  "Commandes de configuration",
}

var initConfigCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialiser la configuration",
	RunE:  runConfigInit,
}

func init() {
	configCmd.AddCommand(initConfigCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = ".code-review-agent.yml"
	}

	content := `# Code Review Agent Configuration
llm:
  provider: mistral
  model: mistral-tiny
  max_tokens: 1024
  temperature: 0.7

cache:
  enabled: true
  dir: ~/.cache/code-review-agent
  ttl: 604800  # 7 days

analysis:
  local_checks: true
  ai_enabled: false
  threshold: 0.5

output:
  format: cli
`

	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("erreur sauvegarde config: %w", err)
	}

	fmt.Printf("Configuration initialisée dans %s\n", cfgPath)
	return nil
}
