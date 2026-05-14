package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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

func initConfigCommands() {
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

	if err := os.WriteFile(cfgPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("erreur sauvegarde config: %w", err)
	}

	fmt.Printf("Configuration initialisée dans %s\n", cfgPath)
	return nil
}
