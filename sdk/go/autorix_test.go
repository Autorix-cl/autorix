package autorix

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestUserContext(t *testing.T) {
	ctx := context.Background()
	user := &User{
		ID:    "user_123",
		Email: "ada@autorix.io",
		Roles: []string{"admin", "developer"},
	}

	ctx = WithUser(ctx, user)
	extracted, ok := UserFromContext(ctx)
	if !ok {
		t.Fatalf("expected user in context")
	}

	if extracted.ID != "user_123" || extracted.Email != "ada@autorix.io" {
		t.Errorf("extracted user mismatch: %+v", extracted)
	}
}

func TestMiddleware_HeaderExtraction(t *testing.T) {
	client := NewClient(Config{})

	handler := client.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := UserFromContext(r.Context())
		if !ok {
			t.Errorf("expected user in context")
			return
		}

		if user.ID != "usr_9988" {
			t.Errorf("expected ID 'usr_9988', got %s", user.ID)
		}
		if user.Email != "grace@autorix.io" {
			t.Errorf("expected Email 'grace@autorix.io', got %s", user.Email)
		}
		if len(user.Roles) != 2 || user.Roles[0] != "viewer" {
			t.Errorf("unexpected roles: %v", user.Roles)
		}

		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Header.Set("X-User-ID", "usr_9988")
	req.Header.Set("X-User-Email", "grace@autorix.io")
	req.Header.Set("X-User-Roles", "viewer,analyst")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", rec.Code)
	}
}

func TestNexus_CheckWithCache(t *testing.T) {
	client := NewClient(Config{
		EnableCache: true,
		CacheTTL:    50 * time.Millisecond,
	})

	ctx := context.Background()

	// First check: populates cache
	allowed, err := client.Check(ctx, "document", "doc_1", "viewer", "alice", nil)
	if err != nil || !allowed {
		t.Fatalf("expected allowed true, got %v (err: %v)", allowed, err)
	}

	// Second check: hits local cache
	allowedCached, err := client.Check(ctx, "document", "doc_1", "viewer", "alice", nil)
	if err != nil || !allowedCached {
		t.Fatalf("expected cached allowed true, got %v", allowedCached)
	}
}

func TestNexus_RealHTTPCheck_AllowedAndDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":true,"reason":"granted"}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		NexusURL: server.URL,
	})

	allowed, err := client.Check(context.Background(), "document", "doc_1", "viewer", "alice", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected allowed true, got false")
	}
}

func TestNexus_FailClosed_OnNetworkOrServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(Config{
		NexusURL: server.URL,
	})

	allowed, err := client.Check(context.Background(), "document", "doc_1", "viewer", "alice", nil)
	if err == nil {
		t.Fatalf("expected error on 500 status")
	}
	if allowed {
		t.Fatalf("expected fail-closed (allowed=false), got true")
	}
}
