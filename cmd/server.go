package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"code-review-agent/internal/server"
	"code-review-agent/internal/storage"
)

var (
	serverPort          int
	serverHost          string
	serverWebhookSecret string
	serverDBPath        string
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Démarrer le serveur HTTP",
	Long:  "Démarre le serveur HTTP avec les endpoints webhook GitHub et analyse on-demand.",
	RunE:  runServer,
}

func initServerCmd() {
	serverCmd.Flags().IntVar(&serverPort, "port", 8080, "Port d'écoute")
	serverCmd.Flags().StringVar(&serverHost, "host", "0.0.0.0", "Adresse d'écoute")
	serverCmd.Flags().StringVar(&serverWebhookSecret, "webhook-secret", "", "Secret GitHub webhook (HMAC-SHA256)")
	serverCmd.Flags().StringVar(&serverDBPath, "db", "", "Chemin SQLite (optionnel)")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Apply CLI flags over loaded config
	if serverPort != 8080 || cfg.Server.Port == 0 {
		cfg.Server.Port = serverPort
	}
	if serverHost != "0.0.0.0" || cfg.Server.Host == "" {
		cfg.Server.Host = serverHost
	}
	if serverWebhookSecret != "" {
		cfg.Server.WebhookSecret = serverWebhookSecret
	}

	// Optional SQLite storage
	var store *storage.Store
	if serverDBPath != "" {
		var err error
		store, err = storage.NewStore(serverDBPath)
		if err != nil {
			log.Printf("Warning: could not open storage at %s: %v (continuing without persistence)", serverDBPath, err)
		} else {
			defer store.Close()
		}
	}

	srv := server.New(*cfg, store)

	// Graceful shutdown on SIGTERM / SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		log.Println("Shutdown signal received, stopping server…")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	return srv.Start()
}
