package autorix

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestNexus_RetryWithJitterOnTransientErrors(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			// First 2 calls fail with 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// 3rd call succeeds
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		NexusURL: server.URL,
		RetryConfig: RetryConfig{
			MaxRetries:    3,
			InitialDelay:  10 * time.Millisecond,
			MaxDelay:      50 * time.Millisecond,
			BackoffFactor: 1.5,
		},
	})

	allowed, err := client.Nexus.Check(context.Background(), CheckRequest{
		Namespace: "documents",
		Object:    "doc_100",
		Relation:  "viewer",
		SubjectID: "alice",
	})

	if err != nil {
		t.Fatalf("expected retry to succeed, got error: %v", err)
	}
	if !allowed {
		t.Fatalf("expected allowed true, got false")
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

func TestNexus_BatchCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()

	client := NewClient(Config{NexusURL: server.URL})

	requests := []CheckRequest{
		{Namespace: "doc", Object: "d1", Relation: "read", SubjectID: "u1"},
		{Namespace: "doc", Object: "d2", Relation: "read", SubjectID: "u1"},
		{Namespace: "doc", Object: "d3", Relation: "write", SubjectID: "u1"},
	}

	results, err := client.Nexus.CheckBatch(context.Background(), requests)
	if err != nil {
		t.Fatalf("BatchCheck failed: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for i, allowed := range results {
		if !allowed {
			t.Errorf("result %d expected true", i)
		}
	}
}

func TestThemis_Evaluate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"all_passed": true,
			"total_evaluated": 1,
			"results": [{"policy_id":"pol_1","policy_name":"MFA","passed":true}]
		}`))
	}))
	defer server.Close()

	client := NewClient(Config{ThemisURL: server.URL})

	resp, err := client.Themis.Evaluate(context.Background(), EvaluatePolicyRequest{
		TenantID: "default",
		Context: map[string]interface{}{
			"auth": map[string]interface{}{"mfa": true},
		},
	})

	if err != nil {
		t.Fatalf("Themis Evaluate failed: %v", err)
	}
	if !resp.AllPassed || resp.TotalEvaluated != 1 {
		t.Fatalf("unexpected themis evaluation outcome: %+v", resp)
	}
}

func TestVulcan_VerifyAndAttenuate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/keys/verify" {
			w.Write([]byte(`{"valid":true,"key_id":"k1","scopes":["read"]}`))
			return
		}
		if r.URL.Path == "/keys/attenuate" {
			w.Write([]byte(`{"attenuated_token":"av_live_attenuated","caveats_applied":["ip = 10.0.0.1"]}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(Config{VulcanURL: server.URL})

	ver, err := client.Vulcan.Verify(context.Background(), "av_live_test", nil)
	if err != nil || !ver.Valid {
		t.Fatalf("expected valid key, got err=%v", err)
	}

	att, err := client.Vulcan.Attenuate(context.Background(), "av_live_test", []string{"ip = 10.0.0.1"})
	if err != nil || att != "av_live_attenuated" {
		t.Fatalf("expected attenuated token, got %s (err: %v)", att, err)
	}
}
