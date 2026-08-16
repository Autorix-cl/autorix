package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autorix/hermes/internal/storage/postgres"
	transport "github.com/autorix/hermes/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Starting Autorix Hermes (SAML 2.0 & SCIM 2.0 Engine)...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Database Connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://autorix:autorix_password@localhost:5432/autorix_hermes?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Configuration
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4477"
	}

	spEntityID := os.Getenv("SP_ENTITY_ID")
	if spEntityID == "" {
		spEntityID = "https://hermes.autorix.io/saml/metadata"
	}

	repo := postgres.NewRepository(pool)
	server := transport.NewServer(repo, baseURL, spEntityID)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4477"
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Autorix Hermes listening on :%s (SAML & SCIM)\n", port)
		log.Printf("SAML SP Metadata: %s/saml/metadata\n", baseURL)
		log.Printf("SCIM 2.0 Base URL: %s/scim/v2\n", baseURL)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down Autorix Hermes gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Forced shutdown error: %v\n", err)
	}
	log.Println("Autorix Hermes stopped.")
}
