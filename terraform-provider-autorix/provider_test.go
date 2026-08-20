package main

import (
	"context"
	"testing"
	"time"
)

func TestProviderClient_Initialization(t *testing.T) {
	client := NewClient(ProviderConfig{
		Endpoint: "http://localhost:4400",
		Token:    "secret-token",
		Timeout:  5 * time.Second,
	})

	if client.config.Endpoint != "http://localhost:4400" {
		t.Errorf("expected endpoint http://localhost:4400, got %s", client.config.Endpoint)
	}

	err := client.Ping(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProviderClient_MissingEndpointFails(t *testing.T) {
	client := NewClient(ProviderConfig{})
	err := client.Ping(context.Background())
	if err == nil {
		t.Fatalf("expected error on missing endpoint")
	}
}
