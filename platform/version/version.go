// Package version holds build-time identity, stamped via -ldflags -X so
// GET /info can report what is actually running (P1-S1-T3). Every engine's
// Dockerfile and Makefile injects these three values at build time; without
// stamping they stay at their "dev" defaults.
package version

var (
	// Version is the engine's semantic version (e.g. "1.4.0").
	Version = "dev"
	// BuildSHA is the git commit the binary was built from.
	BuildSHA = "unknown"
	// BuildTime is the RFC3339 build timestamp.
	BuildTime = "unknown"
)
