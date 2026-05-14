package main

import (
	"fmt"

	"github.com/spf13/cobra"

	cachepkg "code-review-agent/internal/cache"
)

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

func initCacheCmd() {
	cacheCmd.AddCommand(clearCacheCmd)
}

func runCacheClear(cmd *cobra.Command, args []string) error {
	if err := cachepkg.Clear(); err != nil {
		return fmt.Errorf("erreur vidage cache: %w", err)
	}

	fmt.Println("Cache vidé avec succès")
	return nil
}
