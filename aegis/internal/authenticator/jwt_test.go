package authenticator

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWTVerification(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var activeKID atomic.Value
	activeKID.Store("trusted")
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []interface{}{map[string]string{
			"kty": "RSA", "kid": activeKID.Load().(string), "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	}))
	defer jwks.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	remote, err := NewJWKSAuthenticator(ctx, jwks.URL, "issuer", "api", true)
	if err != nil {
		t.Fatal(err)
	}
	pinned, err := NewJWTAuthenticator(&key.PublicKey, "issuer", "api")
	if err != nil {
		t.Fatal(err)
	}
	for _, backend := range []struct {
		name string
		auth *JWTAuthenticator
	}{{"pinned", pinned}, {"jwks", remote}} {
		t.Run(backend.name, func(t *testing.T) {
			for _, tc := range []struct {
				name       string
				mutate     func(jwt.MapClaims, *jwt.Token)
				signingKey interface{}
				method     jwt.SigningMethod
				reject     bool
			}{
				{name: "valid"},
				{name: "forged", signingKey: other, reject: true},
				{name: "expired", mutate: func(c jwt.MapClaims, _ *jwt.Token) { c["exp"] = time.Now().Add(-time.Hour).Unix() }, reject: true},
				{name: "missing expiration", mutate: func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "exp") }, reject: true},
				{name: "wrong issuer", mutate: func(c jwt.MapClaims, _ *jwt.Token) { c["iss"] = "attacker" }, reject: true},
				{name: "missing issuer", mutate: func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "iss") }, reject: true},
				{name: "wrong audience", mutate: func(c jwt.MapClaims, _ *jwt.Token) { c["aud"] = "other" }, reject: true},
				{name: "missing audience", mutate: func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "aud") }, reject: true},
				{name: "missing subject", mutate: func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "sub") }, reject: true},
				{name: "id token", mutate: func(c jwt.MapClaims, _ *jwt.Token) { c["token_use"] = "id_token" }, reject: true},
				{name: "missing token use", mutate: func(c jwt.MapClaims, _ *jwt.Token) { delete(c, "token_use") }, reject: true},
				{name: "future not before", mutate: func(c jwt.MapClaims, _ *jwt.Token) { c["nbf"] = time.Now().Add(time.Hour).Unix() }, reject: true},
				{name: "wrong algorithm", method: jwt.SigningMethodRS512, reject: true},
				{name: "HMAC confusion", method: jwt.SigningMethodHS256, signingKey: []byte("attacker"), reject: true},
				{name: "unsigned", method: jwt.SigningMethodNone, signingKey: jwt.UnsafeAllowNoneSignatureType, reject: true},
				{name: "untrusted header URLs ignored", mutate: func(_ jwt.MapClaims, t *jwt.Token) {
					t.Header["jku"] = "http://untrusted.invalid/keys"
					t.Header["jwk"] = map[string]string{"kid": "evil"}
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					claims := jwt.MapClaims{"sub": "alice", "iss": "issuer", "aud": "api", "exp": time.Now().Add(time.Hour).Unix(), "scope": "read write", "token_use": "access_token"}
					method := tc.method
					if method == nil {
						method = jwt.SigningMethodRS256
					}
					token := jwt.NewWithClaims(method, claims)
					token.Header["kid"] = "trusted"
					if tc.mutate != nil {
						tc.mutate(claims, token)
					}
					var signingKey interface{} = key
					if tc.signingKey != nil {
						signingKey = tc.signingKey
					}
					raw, err := token.SignedString(signingKey)
					if err != nil {
						t.Fatal(err)
					}
					req := httptest.NewRequest("GET", "/", nil)
					req.Header.Set("Authorization", "Bearer "+raw)
					session, err := backend.auth.Authenticate(req, nil)
					if tc.reject {
						if err == nil || session != nil {
							t.Fatal("untrusted token authenticated")
						}
						return
					}
					if err != nil {
						t.Fatal(err)
					}
					if session.Subject != "alice" || len(session.Scopes) != 2 {
						t.Fatalf("bad session: %+v", session)
					}
				})
			}
		})
	}
	t.Run("refreshes rotated key and removes retired key", func(t *testing.T) {
		activeKID.Store("rotated")
		for _, kid := range []string{"rotated", "trusted"} {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "alice", "iss": "issuer", "aud": "api", "exp": time.Now().Add(time.Hour).Unix(), "token_use": "access_token"})
			token.Header["kid"] = kid
			raw, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+raw)
			session, err := remote.Authenticate(req, nil)
			if kid == "rotated" && (err != nil || session == nil) {
				t.Fatalf("rotated key failed: %v", err)
			}
			if kid == "trusted" && (err == nil || session != nil) {
				t.Fatal("retired key accepted after refresh")
			}
		}
	})
	for _, kid := range []string{"unknown", ""} {
		t.Run("reject kid "+kid, func(t *testing.T) {
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"sub": "alice", "iss": "issuer", "aud": "api", "exp": time.Now().Add(time.Hour).Unix(), "token_use": "access_token"})
			token.Header["kid"] = kid
			raw, err := token.SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("Authorization", "Bearer "+raw)
			if s, e := remote.Authenticate(req, nil); e == nil || s != nil {
				t.Fatal("untrusted kid authenticated")
			}
		})
	}
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer forged")
	if s, e := (&JWTAuthenticator{}).Authenticate(req, nil); e == nil || s != nil {
		t.Fatal("zero-value authenticator accepted token")
	}
	if _, e := NewJWTAuthenticator(nil, "issuer", "api"); e == nil {
		t.Fatal("missing key accepted")
	}
	if _, e := NewJWTAuthenticator(&key.PublicKey, "", "api"); e == nil {
		t.Fatal("missing issuer accepted")
	}
	if _, e := NewJWTAuthenticator(&key.PublicKey, "issuer", ""); e == nil {
		t.Fatal("missing audience accepted")
	}
}

func TestJWKSConfigurationFailsClosed(t *testing.T) {
	for _, u := range []string{"", "http://example.com/jwks", "file:///keys", "https://user:pass@example.com/keys", "https://example.com/keys#fragment"} {
		if _, err := NewJWKSAuthenticator(context.Background(), u, "issuer", "api", false); err == nil {
			t.Fatalf("accepted %q", u)
		}
	}
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusOK, http.StatusFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", "http://untrusted.invalid")
				w.WriteHeader(status)
				_, _ = w.Write([]byte("not JSON"))
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if _, err := NewJWKSAuthenticator(ctx, server.URL, "issuer", "api", true); err == nil {
				t.Fatal("bad JWKS endpoint accepted")
			}
		})
	}
}
