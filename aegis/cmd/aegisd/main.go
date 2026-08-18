package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/autorix/aegis/internal/authenticator"
	"github.com/autorix/aegis/internal/authorizer"
	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/mutator"
	"github.com/autorix/aegis/internal/proxy"
	"github.com/autorix/aegis/internal/rule"
	httpTransport "github.com/autorix/aegis/internal/transport/http"
)

func main() {
	log.Println("Starting Autorix Aegis (Zero Trust Access Proxy)...")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Load Security Rules
	rulesPath := os.Getenv("RULES_PATH")
	if rulesPath == "" {
		rulesPath = "rules/default.rules.yaml"
	}

	// The Store owns the rules file end-to-end: it loads and compiles it at
	// boot, serves Match() to the proxy pipeline, and is the read/write
	// backend for the admin API below — every mutation there hot-reloads
	// the pipeline and persists back to this same file.
	store, err := rule.NewStore(rulesPath)
	if err != nil {
		log.Fatalf("Failed to load rules: %v\n", err)
	}

	// 2. Initialize Pipeline Handlers
	authenticators := []core.Authenticator{
		authenticator.NewJWTAuthenticator(nil), // Unverified/local parser or keyFunc
		&authenticator.AnonymousAuthenticator{},
		&authenticator.NoopAuthenticator{},
	}

	authorizers := []core.Authorizer{
		&authorizer.AllowAuthorizer{},
		&authorizer.DenyAuthorizer{},
		authorizer.NewNexusAuthorizer(),
	}

	mutators := []core.Mutator{
		mutator.NewHeaderMutator(),
		&mutator.NoopMutator{},
	}

	// 3. Create Pipeline Proxy Handler
	pipelineProxy := proxy.NewPipelineProxy(store, authenticators, authorizers, mutators)

	// 3b. Admin REST API (console rule management) — separate port from the
	// proxy pipeline, since the proxy's mux is a catch-all reverse proxy.
	adminServer := httpTransport.NewServer(store)
	adminPort := os.Getenv("ADMIN_PORT")
	if adminPort == "" {
		adminPort = "4456"
	}

	adminHTTPServer := &http.Server{
		Addr:         ":" + adminPort,
		Handler:      adminServer.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Autorix Aegis admin API listening on :%s\n", adminPort)
		if err := adminHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Admin server error: %v\n", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "4455" // Default ORY Oathkeeper proxy port
	}

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      pipelineProxy,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Autorix Aegis listening on :%s (Zero Trust Proxy)\n", port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v\n", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down Autorix Aegis gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Forced shutdown error: %v\n", err)
	}
	if err := adminHTTPServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Forced admin server shutdown error: %v\n", err)
	}
	log.Println("Autorix Aegis stopped.")
}
