package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "github.com/autorix/nexus/api/autorix/nexus/v1"
	"github.com/autorix/nexus/internal/engine/caveat"
	"github.com/autorix/nexus/internal/engine/graph"
	"github.com/autorix/nexus/internal/storage/postgres"
	transport "github.com/autorix/nexus/internal/transport/grpc"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	log.Println("Starting Autorix Nexus (Hybrid ReBAC Engine)...")

	ctx := context.Background()

	// 1. Initialize DB Connection from environment (12-Factor App)
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/autorix?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	// 2. Initialize Infrastructure Layer (Repository)
	repo := postgres.NewRepository(pool)

	// 3. Initialize Domain Layer (CEL + Graph)
	celEvaluator, err := caveat.NewCELEvaluator()
	if err != nil {
		log.Fatalf("Failed to initialize CEL Evaluator: %v\n", err)
	}

	resolver := graph.NewResolver(repo, celEvaluator)

	// 4. Initialize Transport Layer (gRPC Server)
	grpcServer := grpc.NewServer()
	nexusServer := transport.NewServer(resolver)

	pb.RegisterNexusServiceServer(grpcServer, nexusServer)
	
	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	// 5. Start listening
	port := os.Getenv("PORT")
	if port == "" {
		port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v\n", port, err)
	}

	log.Printf("Nexus listening on :%s (gRPC)\n", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v\n", err)
	}
}
