package core

import "errors"

// Sentinel errors returned by the Repository implementation and consumed by
// the transport layer to pick the right gRPC/HTTP status, instead of every
// caller string-matching driver errors.
var (
	// ErrNotFound is returned when a lookup by id/slug/hash finds no row.
	ErrNotFound = errors.New("argus: not found")

	// ErrEnrollmentTokenInvalid covers every reason a token cannot be
	// consumed right now: unknown hash, expired, revoked, or already at
	// its uses_allowed ceiling. The reasons are deliberately collapsed —
	// a caller across the trust boundary must not learn which one applies,
	// the same way a login endpoint should not reveal whether a hashed
	// credential was wrong versus unknown.
	ErrEnrollmentTokenInvalid = errors.New("argus: enrollment token invalid, expired, revoked, or exhausted")

	// ErrInvalidCredential is returned when an instance credential fails
	// to validate against the stored (or overlap-window previous) hash.
	ErrInvalidCredential = errors.New("argus: invalid instance credential")
)
