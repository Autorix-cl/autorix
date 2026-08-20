package jwks

import (
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

type KeyManager struct {
	mu          sync.RWMutex
	currentKid  string
	privateKeys map[string]*rsa.PrivateKey
	publicKeys  map[string]*rsa.PublicKey
	keyOrder    []string // newest first
}

func NewKeyManager() (*KeyManager, error) {
	// Generate initial RSA 2048-bit key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	kid := uuid.New().String()
	km := &KeyManager{
		currentKid:  kid,
		privateKeys: make(map[string]*rsa.PrivateKey),
		publicKeys:  make(map[string]*rsa.PublicKey),
		keyOrder:    []string{kid},
	}
	km.privateKeys[kid] = privateKey
	km.publicKeys[kid] = &privateKey.PublicKey

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
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	kid := uuid.New().String()

	km.mu.Lock()
	km.currentKid = kid
	km.privateKeys[kid] = privateKey
	km.publicKeys[kid] = &privateKey.PublicKey
	km.keyOrder = append([]string{kid}, km.keyOrder...)
	km.mu.Unlock()

	jwk := km.jwkFromPublicKey(kid, &privateKey.PublicKey)
	return &jwk, nil
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
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		km.mu.RLock()
		defer km.mu.RUnlock()

		if kid, ok := t.Header["kid"].(string); ok && kid != "" {
			if pubKey, found := km.publicKeys[kid]; found {
				return pubKey, nil
			}
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
