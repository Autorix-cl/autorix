package http

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autorix/ego/internal/core"
	"github.com/autorix/ego/internal/session"
	"github.com/autorix/ego/internal/storage/postgres"
	"github.com/fxamacker/cbor/v2"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type fakeWebAuthnStore struct {
	identities map[uuid.UUID]*core.Identity
	byEmail    map[string]*core.Identity
	sessions   map[string]*core.Session
	creds      map[uuid.UUID][]webauthn.Credential
}

func newFakeWebAuthnStore() *fakeWebAuthnStore {
	return &fakeWebAuthnStore{
		identities: make(map[uuid.UUID]*core.Identity),
		byEmail:    make(map[string]*core.Identity),
		sessions:   make(map[string]*core.Session),
		creds:      make(map[uuid.UUID][]webauthn.Credential),
	}
}

func (f *fakeWebAuthnStore) GetSessionByTokenHash(_ context.Context, tokenHash string) (*core.Session, error) {
	sess, ok := f.sessions[tokenHash]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return sess, nil
}

func (f *fakeWebAuthnStore) GetIdentityByID(_ context.Context, id uuid.UUID) (*core.Identity, error) {
	idn, ok := f.identities[id]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return idn, nil
}

func (f *fakeWebAuthnStore) FindIdentityByIdentifier(_ context.Context, identifier string) (*core.Identity, error) {
	idn, ok := f.byEmail[identifier]
	if !ok {
		return nil, postgres.ErrNotFound
	}
	return idn, nil
}

func (f *fakeWebAuthnStore) GetWebAuthnCredentials(_ context.Context, identityID uuid.UUID) ([]webauthn.Credential, error) {
	list := f.creds[identityID]
	out := make([]webauthn.Credential, len(list))
	copy(out, list)
	return out, nil
}

func (f *fakeWebAuthnStore) SaveWebAuthnCredentials(_ context.Context, identityID uuid.UUID, creds []webauthn.Credential) error {
	cp := make([]webauthn.Credential, len(creds))
	copy(cp, creds)
	f.creds[identityID] = cp
	return nil
}

func (f *fakeWebAuthnStore) CreateSession(_ context.Context, s *core.Session) error {
	f.sessions[s.TokenHash] = s
	return nil
}

// buildSoftwareAttestation builds a real WebAuthn "none" attestation response for an ECDSA P-256 key.
func buildSoftwareAttestation(t *testing.T, priv *ecdsa.PrivateKey, credID []byte, challengeB64, origin, rpID string) json.RawMessage {
	t.Helper()

	clientData, err := json.Marshal(map[string]interface{}{
		"type":      "webauthn.create",
		"challenge": challengeB64,
		"origin":    origin,
	})
	if err != nil {
		t.Fatalf("marshal clientData: %v", err)
	}

	xBytes := priv.X.FillBytes(make([]byte, 32))
	yBytes := priv.Y.FillBytes(make([]byte, 32))
	coseKey, err := cbor.Marshal(map[int]interface{}{
		1:  2,  // kty: EC2
		3:  -7, // alg: ES256
		-1: 1,  // crv: P-256
		-2: xBytes,
		-3: yBytes,
	})
	if err != nil {
		t.Fatalf("marshal COSE key: %v", err)
	}

	rpHash := sha256.Sum256([]byte(rpID))
	var authData bytes.Buffer
	authData.Write(rpHash[:])
	// Flags: UP (0x01) | UV (0x04) | AT (0x40) = 0x45
	authData.WriteByte(0x45)
	_ = binary.Write(&authData, binary.BigEndian, uint32(1)) // signCount = 1
	authData.Write(make([]byte, 16))                         // AAGUID (16 zero bytes)
	_ = binary.Write(&authData, binary.BigEndian, uint16(len(credID)))
	authData.Write(credID)
	authData.Write(coseKey)

	attObj, err := cbor.Marshal(map[string]interface{}{
		"fmt":      "none",
		"attStmt":  map[string]interface{}{},
		"authData": authData.Bytes(),
	})
	if err != nil {
		t.Fatalf("marshal attestationObject: %v", err)
	}

	raw, err := json.Marshal(map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(credID),
		"rawId": base64.RawURLEncoding.EncodeToString(credID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"attestationObject": base64.RawURLEncoding.EncodeToString(attObj),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientData),
		},
	})
	if err != nil {
		t.Fatalf("marshal attestation response: %v", err)
	}
	return raw
}

// buildSoftwareAssertion builds a real WebAuthn login assertion signed by the ECDSA P-256 key.
func buildSoftwareAssertion(t *testing.T, priv *ecdsa.PrivateKey, credID []byte, challengeB64, origin, rpID string, signCount uint32) json.RawMessage {
	t.Helper()

	clientData, err := json.Marshal(map[string]interface{}{
		"type":      "webauthn.get",
		"challenge": challengeB64,
		"origin":    origin,
	})
	if err != nil {
		t.Fatalf("marshal clientData: %v", err)
	}
	clientDataHash := sha256.Sum256(clientData)

	rpHash := sha256.Sum256([]byte(rpID))
	var authData bytes.Buffer
	authData.Write(rpHash[:])
	// Flags: UP (0x01) | UV (0x04) = 0x05
	authData.WriteByte(0x05)
	_ = binary.Write(&authData, binary.BigEndian, signCount)

	signedPayload := append(append([]byte{}, authData.Bytes()...), clientDataHash[:]...)
	digest := sha256.Sum256(signedPayload)
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}

	raw, err := json.Marshal(map[string]interface{}{
		"id":    base64.RawURLEncoding.EncodeToString(credID),
		"rawId": base64.RawURLEncoding.EncodeToString(credID),
		"type":  "public-key",
		"response": map[string]interface{}{
			"authenticatorData": base64.RawURLEncoding.EncodeToString(authData.Bytes()),
			"clientDataJSON":    base64.RawURLEncoding.EncodeToString(clientData),
			"signature":         base64.RawURLEncoding.EncodeToString(sig),
		},
	})
	if err != nil {
		t.Fatalf("marshal assertion response: %v", err)
	}
	return raw
}

func TestWebAuthn_FullRegistrationAndLoginCeremony(t *testing.T) {
	store := newFakeWebAuthnStore()
	sm := session.NewManager(time.Hour)
	srv := NewServer(nil, nil, sm, nil, false)
	srv.waStore = store
	routes := srv.Routes()

	identityID := uuid.New()
	identity := &core.Identity{
		ID:       identityID,
		SchemaID: "default",
		Traits: map[string]interface{}{
			"email": "passkey@autorix.io",
			"name":  map[string]interface{}{"first": "Alice", "last": "Key"},
		},
		State: core.StateActive,
	}
	store.identities[identityID] = identity
	store.byEmail["passkey@autorix.io"] = identity

	// Seed an authenticated session for registration.
	initSess, err := sm.GenerateSession(identityID)
	if err != nil {
		t.Fatalf("GenerateSession: %v", err)
	}
	initSess.Identity = identity
	store.sessions[initSess.TokenHash] = initSess

	// 1. Unauthenticated registration start must be rejected with 401.
	unauthRec := httptest.NewRecorder()
	routes.ServeHTTP(unauthRec, httptest.NewRequest(http.MethodPost, "/self-service/webauthn/registration/start", nil))
	if unauthRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated registration start, got %d", unauthRec.Code)
	}

	// 2. Authenticated registration start returns challenge_id + PublicKeyCredentialCreationOptions.
	regStartReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/registration/start", nil)
	regStartReq.Header.Set("Authorization", "Bearer "+initSess.Token)
	regStartRec := httptest.NewRecorder()
	routes.ServeHTTP(regStartRec, regStartReq)
	if regStartRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on registration start, got %d: %s", regStartRec.Code, regStartRec.Body.String())
	}

	var regStartBody struct {
		ChallengeID string `json:"challenge_id"`
		Options     struct {
			Response struct {
				Challenge string `json:"challenge"`
			} `json:"publicKey"`
		} `json:"options"`
	}
	if err := json.Unmarshal(regStartRec.Body.Bytes(), &regStartBody); err != nil {
		t.Fatalf("decode regStartBody: %v", err)
	}
	if regStartBody.ChallengeID == "" || regStartBody.Options.Response.Challenge == "" {
		t.Fatalf("missing challenge_id or challenge in start response: %s", regStartRec.Body.String())
	}

	// 3. Complete cryptographic registration using software ECDSA P-256 authenticator.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	credID := []byte("autorix-test-passkey-id-01")
	attPayload := buildSoftwareAttestation(t, priv, credID, regStartBody.Options.Response.Challenge, "http://localhost:3000", "localhost")

	regFinishBody, _ := json.Marshal(map[string]interface{}{
		"challenge_id": regStartBody.ChallengeID,
		"credential":   attPayload,
	})
	regFinishReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/registration/finish", bytes.NewReader(regFinishBody))
	regFinishReq.Header.Set("Authorization", "Bearer "+initSess.Token)
	regFinishRec := httptest.NewRecorder()
	routes.ServeHTTP(regFinishRec, regFinishReq)
	if regFinishRec.Code != http.StatusCreated {
		t.Fatalf("expected 201 on registration finish, got %d: %s", regFinishRec.Code, regFinishRec.Body.String())
	}
	if len(store.creds[identityID]) != 1 {
		t.Fatalf("expected 1 persisted passkey, got %d", len(store.creds[identityID]))
	}

	// 4. Replaying the same registration challenge must fail (single-use consumption).
	replayReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/registration/finish", bytes.NewReader(regFinishBody))
	replayReq.Header.Set("Authorization", "Bearer "+initSess.Token)
	replayRec := httptest.NewRecorder()
	routes.ServeHTTP(replayRec, replayReq)
	if replayRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on replayed registration challenge, got %d", replayRec.Code)
	}

	// 5. Start passkey login for passkey@autorix.io.
	loginStartReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/login/start", bytes.NewBufferString(`{"identifier":"passkey@autorix.io"}`))
	loginStartRec := httptest.NewRecorder()
	routes.ServeHTTP(loginStartRec, loginStartReq)
	if loginStartRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on login start, got %d: %s", loginStartRec.Code, loginStartRec.Body.String())
	}

	var loginStartBody struct {
		ChallengeID string `json:"challenge_id"`
		Options     struct {
			Response struct {
				Challenge string `json:"challenge"`
			} `json:"publicKey"`
		} `json:"options"`
	}
	if err := json.Unmarshal(loginStartRec.Body.Bytes(), &loginStartBody); err != nil {
		t.Fatalf("decode loginStartBody: %v", err)
	}

	// 6. Finish passkey login with valid ECDSA assertion (signCount = 2 > 1).
	assertPayload := buildSoftwareAssertion(t, priv, credID, loginStartBody.Options.Response.Challenge, "http://localhost:3000", "localhost", 2)
	loginFinishBody, _ := json.Marshal(map[string]interface{}{
		"challenge_id": loginStartBody.ChallengeID,
		"credential":   assertPayload,
	})
	loginFinishReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/login/finish", bytes.NewReader(loginFinishBody))
	loginFinishRec := httptest.NewRecorder()
	routes.ServeHTTP(loginFinishRec, loginFinishReq)
	if loginFinishRec.Code != http.StatusOK {
		t.Fatalf("expected 200 on login finish, got %d: %s", loginFinishRec.Code, loginFinishRec.Body.String())
	}
	if store.creds[identityID][0].Authenticator.SignCount != 2 {
		t.Fatalf("expected signCount updated to 2, got %d", store.creds[identityID][0].Authenticator.SignCount)
	}

	// Verify session cookie was issued.
	cookies := loginFinishRec.Result().Cookies()
	foundCookie := false
	for _, c := range cookies {
		if c.Name == SessionCookieName && c.Value != "" {
			foundCookie = true
		}
	}
	if !foundCookie {
		t.Fatalf("expected %s cookie on WebAuthn login finish", SessionCookieName)
	}

	// 7. Disabled/suspended identity must be rejected on login start.
	identity.State = core.StateSuspended
	suspendedReq := httptest.NewRequest(http.MethodPost, "/self-service/webauthn/login/start", bytes.NewBufferString(`{"identifier":"passkey@autorix.io"}`))
	suspendedRec := httptest.NewRecorder()
	routes.ServeHTTP(suspendedRec, suspendedReq)
	if suspendedRec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for suspended identity login start, got %d", suspendedRec.Code)
	}
}
