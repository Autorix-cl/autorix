package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/google/uuid"
)

// DefaultSessionLifespan defines active session duration (30 days)
const DefaultSessionLifespan = 30 * 24 * time.Hour

type Manager struct {
	lifespan time.Duration
}

func NewManager(lifespan time.Duration) *Manager {
	if lifespan <= 0 {
		lifespan = DefaultSessionLifespan
	}
	return &Manager{lifespan: lifespan}
}

// GenerateSession creates a new session model with a high-entropy secret token
func (m *Manager) GenerateSession(identityID uuid.UUID) (*core.Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	rawToken := hex.EncodeToString(tokenBytes)
	tokenHash := HashToken(rawToken)

	now := time.Now()
	return &core.Session{
		ID:              uuid.New(),
		IdentityID:      identityID,
		Token:           rawToken,
		TokenHash:       tokenHash,
		ExpiresAt:       now.Add(m.lifespan),
		AuthenticatedAt: now,
	}, nil
}

// HashToken produces a deterministic SHA-256 hash of the bearer token
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
