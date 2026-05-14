package main

import (
	"os"

	"github.com/spf13/cobra"

	"code-review-agent/internal/config"
)

var cfg *config.Config

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

	initAnalyzeCmd()
	initBatchCmd()
	initCacheCmd()
	initConfigCommands()

	rootCmd.AddCommand(analyzeCmd, batchCmd, cacheCmd, configCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
