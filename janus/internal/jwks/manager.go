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
	"time"

	"github.com/autorix/janus/internal/core"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type KeyManager struct {
	kid        string
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
}

func NewKeyManager() (*KeyManager, error) {
	// Generate initial RSA 2048-bit key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	kid := uuid.New().String()
	return &KeyManager{
		kid:        kid,
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
	}, nil
}

func (km *KeyManager) KeyID() string {
	return km.kid
}

// SignJWT creates and signs a JWT token using RS256 with the Key ID in the header
func (km *KeyManager) SignJWT(claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = km.kid

	signedString, err := token.SignedString(km.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signedString, nil
}

// VerifyJWT validates the signature and standard claims of a JWT
func (km *KeyManager) VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return km.publicKey, nil
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

// ExportJWKS converts the public key to standard RFC 7517 JSON Web Key Set
func (km *KeyManager) ExportJWKS() *core.JWKS {
	nBytes := km.publicKey.N.Bytes()
	eBytes := big.NewInt(int64(km.publicKey.E)).Bytes()

	jwk := core.JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: km.kid,
		N:   base64.RawURLEncoding.EncodeToString(nBytes),
		E:   base64.RawURLEncoding.EncodeToString(eBytes),
	}

	return &core.JWKS{
		Keys: []core.JWK{jwk},
	}
}

// ExportPEM returns the PEM encoded private and public keys for persistence
func (km *KeyManager) ExportPEM() (privatePEM, publicPEM string) {
	privBytes := x509.MarshalPKCS1PrivateKey(km.privateKey)
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}
	privatePEM = string(pem.EncodeToMemory(privBlock))

	pubBytes := x509.MarshalPKCS1PublicKey(km.publicKey)
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
