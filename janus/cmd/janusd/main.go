package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autorix/janus/internal/jwks"
	"github.com/autorix/janus/internal/oauth2"
	"github.com/autorix/janus/internal/storage/postgres"
	transport "github.com/autorix/janus/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Starting Autorix Janus (OAuth2 & OIDC Server)...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Database Connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://autorix:autorix_password@localhost:5432/autorix_janus?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Initialize Key Manager & OAuth2 Engine
	issuer := os.Getenv("ISSUER_URL")
	if issuer == "" {
		issuer = "http://localhost:4444"
	}

	keyManager, err := jwks.NewKeyManager()
	if err != nil {
		log.Fatalf("Failed to initialize JWKS Key Manager: %v\n", err)
	}

	engine := oauth2.NewEngine(issuer, keyManager)
	repo := postgres.NewRepository(pool)

	// 3. HTTP Server
	server := transport.NewServer(issuer, repo, keyManager, engine)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4444" // Default ORY Hydra public port
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Autorix Janus listening on :%s (OAuth2/OIDC)\n", port)
		log.Printf("JWKS exposed at %s/.well-known/jwks.json\n", issuer)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down Autorix Janus gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Forced shutdown error: %v\n", err)
	}
	log.Println("Autorix Janus stopped.")
}
