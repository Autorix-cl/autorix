package autorix

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"allowed":true}`))
	}))
	defer server.Close()

	client := NewClient(Config{
		NexusURL:    server.URL,
		EnableCache: true,
		CacheTTL:    50 * time.Millisecond,
	})

	ctx := context.Background()

	// First check: populates cache
	allowed, err := client.Check(ctx, "document", "doc_1", "viewer", "alice", nil)
	if err != nil || !allowed {
		t.Fatalf("expected allowed true, got %v (err: %v)", allowed, err)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 server call, got %d", calls)
	}

	// Second check: hits local cache
	allowedCached, err := client.Check(ctx, "document", "doc_1", "viewer", "alice", nil)
	if err != nil || !allowedCached {
		t.Fatalf("expected cached allowed true, got %v", allowedCached)
	}
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("expected 1 server call (cache hit), got %d", calls)
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
		if r.URL.Path != "/policies/evaluate" {
			t.Fatalf("expected public evaluation route, got %s", r.URL.Path)
		}
		var wire map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&wire); err != nil {
			t.Fatalf("decode wire body: %v", err)
		}
		if _, hasPayload := wire["payload"]; !hasPayload {
			t.Fatalf("expected wire JSON key 'payload', got %+v", wire)
		}
		if _, hasContext := wire["context"]; hasContext {
			t.Fatalf("unexpected legacy JSON key 'context' on wire: %+v", wire)
		}
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

func TestJanus_PublicOAuthOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/openid-configuration":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"issuer":"https://issuer.example","authorization_endpoint":"https://issuer.example/oauth2/auth","token_endpoint":"https://issuer.example/oauth2/token","jwks_uri":"https://issuer.example/.well-known/jwks.json"}`))
		case "/oauth2/token":
			if got := r.FormValue("code_verifier"); got != "verifier" {
				t.Errorf("expected PKCE verifier, got %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"access_token":"access","token_type":"Bearer","expires_in":3600,"refresh_token":"refresh"}`))
		case "/oauth2/revoke":
			user, _, ok := r.BasicAuth()
			if !ok || user != "service" {
				t.Error("expected confidential client authentication")
			}
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(Config{JanusURL: server.URL})
	url, err := client.Janus.AuthorizationURL("browser", "https://app.example/callback", "state", "openid offline_access", "challenge")
	if err != nil || !strings.Contains(url, "code_challenge_method=S256") {
		t.Fatalf("unexpected authorization URL: %q, %v", url, err)
	}
	discovery, err := client.Janus.Discovery(context.Background())
	if err != nil || discovery.Issuer != "https://issuer.example" {
		t.Fatalf("unexpected discovery: %+v, %v", discovery, err)
	}
	token, err := client.Janus.ExchangeToken(context.Background(), TokenRequest{GrantType: "authorization_code", ClientID: "browser", Code: "code", CodeVerifier: "verifier", RedirectURI: "https://app.example/callback"})
	if err != nil || token.RefreshToken != "refresh" {
		t.Fatalf("unexpected token response: %+v, %v", token, err)
	}
	if err := client.Janus.Revoke(context.Background(), "service", "secret", "access", "access_token"); err != nil {
		t.Fatal(err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func TestGoSDK_Hardening_NoSocketRequired(t *testing.T) {
	t.Run("Themis Evaluate sends payload JSON field", func(t *testing.T) {
		c := NewClient(Config{
			ThemisURL: "https://themis.test",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				var body map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("decode: %v", err)
				}
				if _, ok := body["payload"]; !ok {
					t.Fatalf("expected 'payload' key, got %+v", body)
				}
				return jsonResponse(http.StatusOK, `{"all_passed":true,"total_evaluated":1,"results":[]}`), nil
			})},
		})
		resp, err := c.Themis.Evaluate(context.Background(), EvaluatePolicyRequest{
			Context: map[string]interface{}{"amount": 10},
		})
		if err != nil || !resp.AllPassed {
			t.Fatalf("Evaluate failed: resp=%+v err=%v", resp, err)
		}
	})

	t.Run("Nexus cache bypasses context/snap_token/explain, isolates SubjectRelation, bounds size", func(t *testing.T) {
		var calls int32
		c := NewClient(Config{
			NexusURL:        "https://nexus.test",
			EnableCache:     true,
			CacheTTL:        time.Minute,
			MaxCacheEntries: 2,
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				return jsonResponse(http.StatusOK, `{"allowed":true}`), nil
			})},
		})
		ctx := context.Background()
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1"})
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1"}) // hit
		if atomic.LoadInt32(&calls) != 1 {
			t.Fatalf("expected 1 call after cache hit, got %d", calls)
		}
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1", RequestContext: map[string]interface{}{"ip": "1.2.3.4"}})
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1", SnapToken: "zk1"})
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1", Explain: true})
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "1", Relation: "view", SubjectID: "u1", SubjectRelation: "member"})
		if atomic.LoadInt32(&calls) != 5 {
			t.Fatalf("expected 5 calls with bypasses and distinct SubjectRelation, got %d", calls)
		}
		// Exceed MaxCacheEntries=2
		_, _ = c.Nexus.Check(ctx, CheckRequest{Namespace: "doc", Object: "2", Relation: "view", SubjectID: "u2"})
		c.cache.mu.RLock()
		n := len(c.cache.items)
		c.cache.mu.RUnlock()
		if n > 2 {
			t.Fatalf("expected cache bounded to 2 entries, got %d", n)
		}
	})

	t.Run("Nexus Expand escapes query params and checks HTTP status", func(t *testing.T) {
		var rawQuery string
		c := NewClient(Config{
			NexusURL: "https://nexus.test",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				rawQuery = r.URL.RawQuery
				return jsonResponse(http.StatusBadGateway, `{"error":"down"}`), nil
			})},
		})
		_, err := c.Nexus.Expand(context.Background(), "doc&x=1", "obj/2", "view#3")
		if err == nil {
			t.Fatalf("expected error on 502 status from Expand")
		}
		if !strings.Contains(rawQuery, "namespace=doc%26x%3D1") {
			t.Fatalf("expected escaped namespace query param, got %q", rawQuery)
		}
	})

	t.Run("Nexus LookupResources checks HTTP status", func(t *testing.T) {
		c := NewClient(Config{
			NexusURL: "https://nexus.test",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusInternalServerError, `{"error":"boom"}`), nil
			})},
		})
		if _, err := c.Nexus.LookupResources(context.Background(), "doc", "view", "u1", "user"); err == nil {
			t.Fatalf("expected error on 500 from LookupResources")
		}
	})

	t.Run("Vulcan VerifyMacaroon & AttenuateMacaroon wire contract and status check", func(t *testing.T) {
		c := NewClient(Config{
			VulcanURL:   "https://vulcan.test",
			RetryConfig: RetryConfig{MaxRetries: 0},
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				switch r.URL.Path {
				case "/keys/verify":
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if _, ok := body["macaroon"]; !ok {
						return jsonResponse(http.StatusBadRequest, `{"error":"macaroon is required"}`), nil
					}
					return jsonResponse(http.StatusOK, `{"valid":true,"api_key":{"id":"k-9","name":"prod","scopes":["read"]}}`), nil
				case "/keys/attenuate":
					var body map[string]interface{}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body["caveat"] != "method = GET" {
						return jsonResponse(http.StatusBadRequest, `{"error":"missing caveat"}`), nil
					}
					return jsonResponse(http.StatusOK, `{"location":"https://api.test","key_id":"k-9","caveats":[{"predicate":"method = GET"}],"signature":"sig2"}`), nil
				}
				return jsonResponse(http.StatusNotFound, `{}`), nil
			})},
		})
		mac := &Macaroon{Location: "https://api.test", KeyID: "k-9", Signature: "sig1"}
		ver, err := c.Vulcan.VerifyMacaroon(context.Background(), mac, nil)
		if err != nil || !ver.Valid || ver.KeyID != "k-9" || ver.Name != "prod" {
			t.Fatalf("VerifyMacaroon failed: ver=%+v err=%v", ver, err)
		}
		att, err := c.Vulcan.AttenuateMacaroon(context.Background(), mac, "method = GET")
		if err != nil || len(att.Caveats) != 1 || att.Signature != "sig2" {
			t.Fatalf("AttenuateMacaroon failed: att=%+v err=%v", att, err)
		}

		// Status 500 must return an error
		cErr := NewClient(Config{
			VulcanURL:   "https://vulcan.test",
			RetryConfig: RetryConfig{MaxRetries: 0},
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusInternalServerError, `{"error":"db down"}`), nil
			})},
		})
		if _, err := cErr.Vulcan.VerifyMacaroon(context.Background(), mac, nil); err == nil {
			t.Fatalf("expected error on 500 from Vulcan Verify")
		}
	})

	t.Run("Janus GetJWKS checks status and returns defensive copy, ExchangeToken omits APIKey", func(t *testing.T) {
		var jwksCalls int32
		c := NewClient(Config{
			JanusURL: "https://janus.test",
			APIKey:   "av_live_secret",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.URL.Path == "/.well-known/jwks.json" {
					atomic.AddInt32(&jwksCalls, 1)
					return jsonResponse(http.StatusOK, `{"keys":[{"kid":"k1"}]}`), nil
				}
				if r.URL.Path == "/oauth2/token" {
					if r.Header.Get("Authorization") != "" {
						t.Errorf("expected no Authorization header on public PKCE call, got %q", r.Header.Get("Authorization"))
					}
					return jsonResponse(http.StatusOK, `{"access_token":"at","token_type":"Bearer","expires_in":60}`), nil
				}
				return jsonResponse(http.StatusInternalServerError, `{}`), nil
			})},
		})

		jwks1, err := c.Janus.GetJWKS(context.Background())
		if err != nil {
			t.Fatalf("GetJWKS 1: %v", err)
		}
		// Mutate returned map; cached copy must remain intact
		jwks1["poison"] = true
		jwks2, err := c.Janus.GetJWKS(context.Background())
		if err != nil {
			t.Fatalf("GetJWKS 2: %v", err)
		}
		if _, leaked := jwks2["poison"]; leaked {
			t.Fatalf("mutation of returned JWKS map leaked into cache")
		}
		if atomic.LoadInt32(&jwksCalls) != 1 {
			t.Fatalf("expected 1 JWKS network call, got %d", jwksCalls)
		}

		if _, err := c.Janus.ExchangeToken(context.Background(), TokenRequest{
			GrantType: "authorization_code", ClientID: "spa", Code: "c", CodeVerifier: "v",
		}); err != nil {
			t.Fatalf("ExchangeToken: %v", err)
		}

		// Non-200 JWKS must return error
		cBad := NewClient(Config{
			JanusURL: "https://janus.test",
			HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return jsonResponse(http.StatusServiceUnavailable, `{}`), nil
			})},
		})
		if _, err := cBad.Janus.GetJWKS(context.Background()); err == nil {
			t.Fatalf("expected error on 503 from GetJWKS")
		}
	})
}
