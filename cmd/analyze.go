package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"code-review-agent/internal/aggregator"
	"code-review-agent/internal/analyzer/llm"
	"code-review-agent/internal/analyzer/local"
	"code-review-agent/internal/formatter"
	"code-review-agent/internal/models"
	"code-review-agent/internal/parser"
)

var (
	analyzeFile   string
	analyzeStdin  bool
	analyzeLLM    bool
	analyzeOutput string
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyser un diff",
	Long:  "Analyser un diff pour détecter les problèmes de code",
	RunE:  runAnalyze,
}

func initAnalyzeCmd() {
	analyzeCmd.Flags().StringVar(&analyzeFile, "file", "", "Chemin du fichier diff")
	analyzeCmd.Flags().BoolVar(&analyzeStdin, "stdin", false, "Lire depuis stdin")
	analyzeCmd.Flags().BoolVar(&analyzeLLM, "llm", false, "Activer analyse LLM")
	analyzeCmd.Flags().StringVar(&analyzeOutput, "output", "", "Fichier de sortie (défaut: stdout)")
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	// 1. Lire le diff
	var diffContent string
	if analyzeFile != "" {
		if !isSecurePath(analyzeFile) {
			return fmt.Errorf("invalid file path: %s", analyzeFile)
		}
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
	h := sha256.Sum256([]byte(diffContent))
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
		if err := os.WriteFile(analyzeOutput, []byte(output), 0600); err != nil {
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
