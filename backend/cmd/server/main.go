package main

import (
	"context"
	"log"

	"github.com/dimitris-taskou/cost-calculator/internal/config"
	fb "github.com/dimitris-taskou/cost-calculator/internal/firebase"
	"github.com/dimitris-taskou/cost-calculator/internal/router"
	"github.com/dimitris-taskou/cost-calculator/internal/seed"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	client, err := fb.New(ctx, cfg.CredentialsFile, cfg.FirebaseProject)
	if err != nil {
		log.Fatalf("firebase init failed: %v", err)
	}
	defer client.Firestore.Close()

	if err := seed.Providers(ctx, client.Firestore); err != nil {
		log.Printf("seed warning: %v", err)
	}
	seed.MigrateExpenseMonths(ctx, client.Firestore)

	r := router.New(client.Firestore, client.Auth, client.App, cfg.AnthropicAPIKey, cfg.SchedulerSecret, cfg.NordigenSecretID, cfg.NordigenSecretKey, cfg.NordigenRedirectURL)

	log.Printf("server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
