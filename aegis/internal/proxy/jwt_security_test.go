package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autorix/aegis/internal/authenticator"
	"github.com/autorix/aegis/internal/authorizer"
	"github.com/autorix/aegis/internal/core"
	"github.com/autorix/aegis/internal/rule"
	"github.com/golang-jwt/jwt/v5"
)

func TestPipelineJWTRejectsForgeryBeforeUpstream(t *testing.T) {
	trusted, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	attacker, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var hits atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1); w.WriteHeader(http.StatusOK) }))
	defer backend.Close()
	matcher, err := rule.NewMatcherFromYAML([]byte(fmt.Sprintf(`
- id: protected
  match:
    url: /protected
    methods: [GET]
  authenticators:
    - handler: jwt
  authorizer:
    handler: allow
  upstream:
    url: "%s"
`, backend.URL)))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := authenticator.NewJWTAuthenticator(&trusted.PublicKey, "issuer", "api")
	if err != nil {
		t.Fatal(err)
	}
	pipeline := NewPipelineProxy(matcher, []core.Authenticator{auth}, []core.Authorizer{&authorizer.AllowAuthorizer{}}, nil)
	for _, tc := range []struct {
		name   string
		key    *rsa.PrivateKey
		status int
		hits   int32
	}{
		{"forged", attacker, http.StatusUnauthorized, 0},
		{"valid", trusted, http.StatusOK, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "alice", "iss": "issuer", "aud": "api", "exp": time.Now().Add(time.Hour).Unix(), "token_use": "access_token"})
			raw, err := token.SignedString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest("GET", "/protected", nil)
			request.Header.Set("Authorization", "Bearer "+raw)
			recorder := httptest.NewRecorder()
			pipeline.ServeHTTP(recorder, request)
			if recorder.Code != tc.status || hits.Load() != tc.hits {
				t.Fatalf("status=%d upstream hits=%d", recorder.Code, hits.Load())
			}
		})
	}
}
