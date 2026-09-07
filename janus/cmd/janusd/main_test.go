package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/autorix/platform/config"
	"github.com/autorix/platform/run"
)

func TestAdminListenerDefaultsToLoopback(t *testing.T) {
	// The config loader distinguishes unset from empty; clear inherited values.
	for _, key := range []string{"ADMIN_HOST", "ADMIN_PORT"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	var cfg appConfig
	if err := config.Load(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.AdminHost != "127.0.0.1" || cfg.AdminPort != "4445" {
		t.Fatalf("unsafe admin defaults: %s:%s", cfg.AdminHost, cfg.AdminPort)
	}
}

func TestCORSAllowedOriginsFailsClosedByDefault(t *testing.T) {
	for _, tt := range []struct {
		raw  string
		want []string
	}{
		{raw: "", want: []string{}},
		{raw: " https://console.example , https://app.example ", want: []string{"https://console.example", "https://app.example"}},
	} {
		got := corsAllowedOrigins(tt.raw)
		if len(got) != len(tt.want) {
			t.Fatalf("corsAllowedOrigins(%q) = %v, want %v", tt.raw, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("corsAllowedOrigins(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		}
	}
}

func TestBindFailureClosesPreviouslyBoundListener(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = occupied.Close() }()
	available, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	firstAddress := available.Addr().String()
	_ = available.Close()
	first := &http.Server{ReadHeaderTimeout: time.Second, Addr: firstAddress}
	if listeners, err := bindHTTPServers(first, &http.Server{ReadHeaderTimeout: time.Second, Addr: occupied.Addr().String()}); err == nil || listeners != nil {
		t.Fatalf("bind must fail atomically: listeners=%v err=%v", listeners, err)
	}
	rebound, err := net.Listen("tcp", firstAddress)
	if err != nil {
		t.Fatalf("first listener leaked after second bind failed: %v", err)
	}
	_ = rebound.Close()
}

func TestBothHTTPListenersServeAndShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	public := &http.Server{ReadHeaderTimeout: time.Second, Addr: "127.0.0.1:0", Handler: handler}
	admin := &http.Server{ReadHeaderTimeout: time.Second, Addr: "127.0.0.1:0", Handler: handler}
	listeners, err := bindHTTPServers(public, admin)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		finished <- run.Run(ctx, time.Second, nil, []run.Named{
			httpRunner("public", public, listeners[0]), httpRunner("admin", admin, listeners[1]),
		})
	}()
	client := &http.Client{Timeout: time.Second}
	for _, listener := range listeners {
		resp, err := client.Get("http://" + listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("unexpected status: %d", resp.StatusCode)
		}
	}
	cancel()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("both listeners failed to stop")
	}
	for _, listener := range listeners {
		conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
		if err == nil {
			_ = conn.Close()
			t.Fatal("listener remains open after shutdown")
		}
	}
}
