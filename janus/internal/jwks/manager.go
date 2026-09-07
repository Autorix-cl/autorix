package jwks

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// KeyStore provides durable key lifecycle operations. It is intentionally
// small so the key manager does not depend on a concrete database package.
type KeyStore interface {
	InitializeSigningKey(context.Context, core.SigningKey) ([]core.SigningKey, error)
	RotateSigningKey(context.Context, core.SigningKey) ([]core.SigningKey, error)
	ListSigningKeys(context.Context) ([]core.SigningKey, error)
}

type KeyManager struct {
	mu          sync.RWMutex
	currentKid  string
	privateKeys map[string]*rsa.PrivateKey
	publicKeys  map[string]*rsa.PublicKey
	keyOrder    []string // newest first
	store       KeyStore
}

func NewKeyManager() (*KeyManager, error) {
	key, privateKey, err := generateSigningKey()
	if err != nil {
		return nil, err
	}
	km := &KeyManager{
		currentKid:  key.Kid,
		privateKeys: make(map[string]*rsa.PrivateKey),
		publicKeys:  make(map[string]*rsa.PublicKey),
		keyOrder:    []string{key.Kid},
	}
	km.privateKeys[key.Kid] = privateKey
	km.publicKeys[key.Kid] = &privateKey.PublicKey

	return km, nil
}

// NewPersistentKeyManager loads the cluster-wide signing-key set from the
// durable store, atomically creating the first key when none exists. It must
// be used by service startup; NewKeyManager remains for isolated unit tests.
func NewPersistentKeyManager(ctx context.Context, store KeyStore) (*KeyManager, error) {
	if store == nil {
		return nil, errors.New("signing key store is required")
	}
	candidate, _, err := generateSigningKey()
	if err != nil {
		return nil, err
	}
	keys, err := store.InitializeSigningKey(ctx, candidate)
	if err != nil {
		return nil, fmt.Errorf("initialize persistent signing keys: %w", err)
	}
	km := &KeyManager{store: store}
	if err := km.replaceKeys(keys); err != nil {
		return nil, err
	}
	return km, nil
}

func (km *KeyManager) KeyID() string {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.currentKid
}

// KeyCount returns the number of active/known JWKS keys.
func (km *KeyManager) KeyCount() int {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return len(km.publicKeys)
}

// RotateKey generates a new RSA key pair, sets it as the current signing key,
// and retains previous keys for verification rollover.
func (km *KeyManager) RotateKey() (*core.JWK, error) {
	key, privateKey, err := generateSigningKey()
	if err != nil {
		return nil, err
	}
	if km.store != nil {
		keys, err := km.store.RotateSigningKey(context.Background(), key)
		if err != nil {
			return nil, fmt.Errorf("persist rotated signing key: %w", err)
		}
		if err := km.replaceKeys(keys); err != nil {
			return nil, err
		}
		km.mu.RLock()
		current := km.currentKid
		public := km.publicKeys[current]
		km.mu.RUnlock()
		jwk := km.jwkFromPublicKey(current, public)
		return &jwk, nil
	}

	km.mu.Lock()
	km.currentKid = key.Kid
	km.privateKeys[key.Kid] = privateKey
	km.publicKeys[key.Kid] = &privateKey.PublicKey
	km.keyOrder = append([]string{key.Kid}, km.keyOrder...)
	km.mu.Unlock()

	jwk := km.jwkFromPublicKey(key.Kid, &privateKey.PublicKey)
	return &jwk, nil
}

// Reload refreshes a persistent manager after a key rotation performed by a
// different Janus instance. It is a no-op for an in-memory test manager.
func (km *KeyManager) Reload(ctx context.Context) error {
	if km.store == nil {
		return nil
	}
	keys, err := km.store.ListSigningKeys(ctx)
	if err != nil {
		return fmt.Errorf("load persistent signing keys: %w", err)
	}
	return km.replaceKeys(keys)
}

// SignJWT creates and signs a JWT token using RS256 with the Key ID in the header
func (km *KeyManager) SignJWT(claims jwt.MapClaims) (string, error) {
	km.mu.RLock()
	currentKid := km.currentKid
	privKey := km.privateKeys[currentKid]
	km.mu.RUnlock()

	if privKey == nil {
		return "", errors.New("no active signing key found")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = currentKid

	signedString, err := token.SignedString(privKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedString, nil
}

// VerifyJWT validates the signature and standard claims of a JWT, checking
// across active and retiring verification keys.
func (km *KeyManager) VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method != jwt.SigningMethodRS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		km.mu.RLock()
		defer km.mu.RUnlock()

		if kid, ok := t.Header["kid"].(string); ok && kid != "" {
			if pubKey, found := km.publicKeys[kid]; found {
				return pubKey, nil
			}
			return nil, errors.New("key not found for token verification")
		}

		// If kid is missing or not found, try current key
		if currentPub, found := km.publicKeys[km.currentKid]; found {
			return currentPub, nil
		}

		return nil, errors.New("key not found for token verification")
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func generateSigningKey() (core.SigningKey, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return core.SigningKey{}, nil, fmt.Errorf("generate RSA signing key: %w", err)
	}
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	pubBlock := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&privateKey.PublicKey)}
	return core.SigningKey{
		Kid:           uuid.NewString(),
		Algorithm:     "RS256",
		Use:           "sig",
		PrivateKeyPEM: string(pem.EncodeToMemory(privBlock)),
		PublicKeyPEM:  string(pem.EncodeToMemory(pubBlock)),
	}, privateKey, nil
}

func (km *KeyManager) replaceKeys(keys []core.SigningKey) error {
	if len(keys) == 0 {
		return errors.New("persistent signing key set is empty")
	}
	privateKeys := make(map[string]*rsa.PrivateKey, 1)
	publicKeys := make(map[string]*rsa.PublicKey, len(keys))
	keyOrder := make([]string, 0, len(keys))
	for i, key := range keys {
		private, public, err := parseSigningKey(key)
		if err != nil {
			return fmt.Errorf("load signing key %q: %w", key.Kid, err)
		}
		if i == 0 {
			privateKeys[key.Kid] = private
		}
		publicKeys[key.Kid] = public
		keyOrder = append(keyOrder, key.Kid)
	}
	km.mu.Lock()
	km.currentKid = keyOrder[0]
	km.privateKeys = privateKeys
	km.publicKeys = publicKeys
	km.keyOrder = keyOrder
	km.mu.Unlock()
	return nil
}

func parseSigningKey(key core.SigningKey) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if key.Kid == "" || key.Algorithm != "RS256" || key.Use != "sig" {
		return nil, nil, errors.New("unsupported key metadata")
	}
	privBlock, rest := pem.Decode([]byte(key.PrivateKeyPEM))
	if privBlock == nil || len(rest) != 0 || privBlock.Type != "RSA PRIVATE KEY" {
		return nil, nil, errors.New("invalid RSA private key PEM")
	}
	private, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil || private.N.BitLen() < 2048 {
		return nil, nil, errors.New("invalid or weak RSA private key")
	}
	pubBlock, rest := pem.Decode([]byte(key.PublicKeyPEM))
	if pubBlock == nil || len(rest) != 0 || pubBlock.Type != "RSA PUBLIC KEY" {
		return nil, nil, errors.New("invalid RSA public key PEM")
	}
	public, err := x509.ParsePKCS1PublicKey(pubBlock.Bytes)
	if err != nil || public.N.BitLen() < 2048 || public.N.Cmp(private.N) != 0 || public.E != private.E {
		return nil, nil, errors.New("public key does not match private key")
	}
	return private, public, nil
}

func (km *KeyManager) jwkFromPublicKey(kid string, pub *rsa.PublicKey) core.JWK {
	nBytes := pub.N.Bytes()
	eBytes := big.NewInt(int64(pub.E)).Bytes()

	return core.JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   base64.RawURLEncoding.EncodeToString(nBytes),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}
}

// ExportJWKS converts the public keys to standard RFC 7517 JSON Web Key Set
func (km *KeyManager) ExportJWKS() *core.JWKS {
	km.mu.RLock()
	defer km.mu.RUnlock()

	keys := make([]core.JWK, 0, len(km.keyOrder))
	for _, kid := range km.keyOrder {
		if pub, exists := km.publicKeys[kid]; exists {
			keys = append(keys, km.jwkFromPublicKey(kid, pub))
		}
	}

	return &core.JWKS{
		Keys: keys,
	}
}

// ExportPEM returns the PEM encoded private and public keys for persistence
func (km *KeyManager) ExportPEM() (privatePEM, publicPEM string) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	privKey := km.privateKeys[km.currentKid]
	pubKey := km.publicKeys[km.currentKid]

	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}
	privatePEM = string(pem.EncodeToMemory(privBlock))

	pubBytes := x509.MarshalPKCS1PublicKey(pubKey)
	pubBlock := &pem.Block{Type: "RSA PUBLIC KEY", Bytes: pubBytes}
	publicPEM = string(pem.EncodeToMemory(pubBlock))

	return privatePEM, publicPEM
}

// GenerateClaims helper to build RFC 7519 compliant claims
func GenerateClaims(issuer, subject, audience string, scopes []string, lifespan time.Duration) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"iss":    issuer,
		"sub":    subject,
		"aud":    audience,
		"iat":    now.Unix(),
		"exp":    now.Add(lifespan).Unix(),
		"jti":    uuid.New().String(),
		"scopes": scopes,
	}
}
