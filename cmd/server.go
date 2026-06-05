package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"

	"code-review-agent/internal/server"
	"code-review-agent/internal/storage"
)

var (
	serverPort          int
	serverHost          string
	serverWebhookSecret string
	serverDBPath        string
	serverGitHubAppID   string
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
	serverCmd.Flags().StringVar(&serverGitHubAppID, "github-app-id", "", "GitHub App ID (authentification App)")
}

func ensureAdminUser(store *storage.Store, adminPassword string) {
	password := adminPassword
	if password == "" {
		password = "changeme"
		log.Println("WARN: no ADMIN_PASSWORD set, using default password 'changeme' — change it immediately!")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash admin password: %v", err)
		return
	}

	count, err := store.UserCount()
	if err != nil {
		log.Printf("failed to count users: %v", err)
		return
	}

	if count == 0 {
		if _, err := store.CreateUser("admin", string(hash), "admin"); err != nil {
			log.Printf("failed to create admin user: %v", err)
			return
		}
		log.Println("Admin user created (username: admin)")
		return
	}

	// Admin already exists — update password if ADMIN_PASSWORD env var is set
	if adminPassword != "" {
		if err := store.UpdateUserPassword("admin", string(hash)); err != nil {
			log.Printf("failed to update admin password: %v", err)
			return
		}
		log.Println("Admin password updated from ADMIN_PASSWORD env var")
	}
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
	if serverGitHubAppID != "" {
		cfg.GitHub.AppID = serverGitHubAppID
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
			ensureAdminUser(store, cfg.Server.AdminPassword)
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
