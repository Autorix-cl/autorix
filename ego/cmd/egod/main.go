package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autorix/ego/internal/credential"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
	transport "github.com/autorix/ego/internal/transport/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Starting Autorix Ego (Identity Engine)...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Database Connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://autorix:autorix_password@localhost:5432/autorix_ego?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Initialize Domain & Storage
	repo := postgres.NewRepository(pool)
	hasher := credential.NewHasher(nil)
	sessionManager := session.NewManager(30 * 24 * time.Hour)

	// 3. HTTP Server
	server := transport.NewServer(repo, hasher, sessionManager)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4433" // Default ORY Kratos public port convention
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      server.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Autorix Ego listening on :%s (HTTP REST)\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down Autorix Ego gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Forced shutdown error: %v\n", err)
	}
	log.Println("Autorix Ego stopped.")
}
