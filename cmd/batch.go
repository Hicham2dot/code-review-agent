package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"code-review-agent/internal/aggregator"
	"code-review-agent/internal/analyzer/llm"
	"code-review-agent/internal/analyzer/local"
	"code-review-agent/internal/formatter"
	"code-review-agent/internal/models"
	"code-review-agent/internal/parser"
)

var (
	batchDir    string
	batchOutput string
)

var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Analyser plusieurs diffs",
	Long:  "Analyser plusieurs fichiers .diff dans un répertoire",
	RunE:  runBatch,
}

func initBatchCmd() {
	batchCmd.Flags().StringVar(&batchDir, "dir", "./diffs", "Répertoire contenant les fichiers .diff")
	batchCmd.Flags().StringVar(&batchOutput, "output", "", "Fichier de sortie (défaut: stdout)")
}

func runBatch(cmd *cobra.Command, args []string) error {
	// 1. Valider le répertoire
	if !isSecurePath(batchDir) {
		return fmt.Errorf("invalid directory path: %s", batchDir)
	}
	if _, err := os.Stat(batchDir); err != nil {
		return fmt.Errorf("répertoire inaccessible: %w", err)
	}

	// 2. Globber les fichiers .diff
	pattern := filepath.Join(batchDir, "*.diff")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("erreur globbing: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Trouvé %d fichiers .diff\n", len(files))
	}

	// 3. Traiter chaque fichier
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
			llmIssues, err = llm.LLMAnalyze(hunks, cfg.LLM)
			if err != nil && verbose {
				fmt.Fprintf(os.Stderr, "Avertissement: analyse LLM échouée pour %s: %v\n", filePath, err)
			}
		}

		// Agrégation
		result := aggregator.Aggregate(localIssues, llmIssues, hunks, diffContent)
		h := sha256.Sum256([]byte(diffContent))
		result.DiffHash = fmt.Sprintf("%x", h)
		batchResults = append(batchResults, result)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Traitement batch complété (%d fichiers)\n", len(batchResults))
	}

	// 4. Formatter et afficher résultats
	output := "# Résultats Batch Analysis\n\n"
	for i, result := range batchResults {
		output += fmt.Sprintf("## Fichier %d (Hash: %s)\n", i+1, result.DiffHash)
		output += formatter.FormatMarkdown(&result)
		output += "\n---\n\n"
	}

	// 5. Afficher ou sauvegarder
	if batchOutput != "" {
		if err := os.WriteFile(batchOutput, []byte(output), 0600); err != nil {
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
