package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProvider_CRUDAndHeaders(t *testing.T) {
	envs := map[string]Environment{}
	policies := map[string]Policy{}
	clients := map[string]OAuth2Client{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/ready" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer tf-secret" || r.Header.Get("X-Autorix-Internal-Token") != "tf-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/environments":
			var env Environment
			if err := json.NewDecoder(r.Body).Decode(&env); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			env.ID = "env_prod"
			envs[env.ID] = env
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(env)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/environments/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/environments/")
			env, ok := envs[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(env)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/environments/"):
			id := strings.TrimPrefix(r.URL.Path, "/v1/environments/")
			delete(envs, id)
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == "/policies":
			var pol Policy
			if err := json.NewDecoder(r.Body).Decode(&pol); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if pol.ID == "" {
				pol.ID = "pol_1"
			}
			policies[pol.ID] = pol
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(pol)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/policies/"):
			id := strings.TrimPrefix(r.URL.Path, "/policies/")
			pol, ok := policies[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(pol)

		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/policies/"):
			id := strings.TrimPrefix(r.URL.Path, "/policies/")
			var pol Policy
			if err := json.NewDecoder(r.Body).Decode(&pol); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			pol.ID = id
			policies[id] = pol
			_ = json.NewEncoder(w).Encode(pol)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/policies/"):
			id := strings.TrimPrefix(r.URL.Path, "/policies/")
			delete(policies, id)
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && r.URL.Path == "/admin/clients":
			var cl OAuth2Client
			if err := json.NewDecoder(r.Body).Decode(&cl); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			cl.ClientSecret = "generated-secret" //nolint:gosec // unit test fixture
			clients[cl.ClientID] = cl
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(cl) //nolint:gosec // mock OAuth2 client creation response

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/admin/clients/"):
			id := strings.TrimPrefix(r.URL.Path, "/admin/clients/")
			cl, ok := clients[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(cl) //nolint:gosec // mock OAuth2 client response

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/admin/clients/"):
			id := strings.TrimPrefix(r.URL.Path, "/admin/clients/")
			delete(clients, id)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p, err := NewProvider(Config{Endpoint: srv.URL, APIToken: "tf-secret"})
	if err != nil {
		t.Fatalf("NewProvider failed: %v", err)
	}
	ctx := context.Background()

	// Ping
	if err := p.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Environment CRUD
	createdEnv, err := p.CreateEnvironment(ctx, Environment{Name: "production", Slug: "prod", Tier: "production"})
	if err != nil || createdEnv.ID != "env_prod" {
		t.Fatalf("CreateEnvironment failed: env=%+v err=%v", createdEnv, err)
	}
	gotEnv, err := p.GetEnvironment(ctx, "env_prod")
	if err != nil || gotEnv.Name != "production" {
		t.Fatalf("GetEnvironment failed: env=%+v err=%v", gotEnv, err)
	}
	if err := p.DeleteEnvironment(ctx, "env_prod"); err != nil {
		t.Fatalf("DeleteEnvironment failed: %v", err)
	}

	// Policy CRUD + idempotent ApplyPolicy
	createdPol, err := p.ApplyPolicy(ctx, Policy{TenantID: "default", Name: "require-mfa", Expression: "request.mfa == true", Active: true})
	if err != nil || createdPol.ID != "pol_1" {
		t.Fatalf("ApplyPolicy (create) failed: pol=%+v err=%v", createdPol, err)
	}
	updatedPol, err := p.ApplyPolicy(ctx, Policy{ID: "pol_1", TenantID: "default", Name: "require-mfa", Expression: "request.mfa == true && request.risk < 50", Active: true})
	if err != nil || !strings.Contains(updatedPol.Expression, "risk < 50") {
		t.Fatalf("ApplyPolicy (update) failed: pol=%+v err=%v", updatedPol, err)
	}
	if err := p.DeletePolicy(ctx, "pol_1"); err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}

	// OAuth2Client CRUD
	createdClient, err := p.CreateOAuth2Client(ctx, OAuth2Client{ClientID: "svc-billing", ClientName: "Billing Service", GrantTypes: []string{"client_credentials"}})
	if err != nil || createdClient.ClientSecret != "generated-secret" {
		t.Fatalf("CreateOAuth2Client failed: cl=%+v err=%v", createdClient, err)
	}
	gotClient, err := p.GetOAuth2Client(ctx, "svc-billing")
	if err != nil || gotClient.ClientName != "Billing Service" {
		t.Fatalf("GetOAuth2Client failed: cl=%+v err=%v", gotClient, err)
	}
	if err := p.DeleteOAuth2Client(ctx, "svc-billing"); err != nil {
		t.Fatalf("DeleteOAuth2Client failed: %v", err)
	}

	// Unauthorized error returns typed *APIError
	badProvider, _ := NewProvider(Config{Endpoint: srv.URL, APIToken: "wrong"})
	_, err = badProvider.GetEnvironment(ctx, "env_prod")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 APIError, got %v", err)
	}
}

func TestRunCLI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health/ready" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var out, errOut bytes.Buffer
	if code := runCLI([]string{"-version"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "terraform-provider-autorix") {
		t.Fatalf("expected version output, code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	if code := runCLI([]string{"-schema"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "autorix_policy") {
		t.Fatalf("expected schema JSON, code=%d out=%q err=%q", code, out.String(), errOut.String())
	}

	out.Reset()
	if code := runCLI([]string{"-check", "-endpoint", srv.URL}, &out, &errOut); code != 0 || strings.TrimSpace(out.String()) != "ok" {
		t.Fatalf("expected check ok, code=%d out=%q err=%q", code, out.String(), errOut.String())
	}
}
