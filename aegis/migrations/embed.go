// Package migrations contains the versioned Aegis database migrations.
//
// The SQL files are embedded so the exact schema that a given aegisd binary
// expects travels with that binary.  Deployments must invoke `aegisd migrate`
// explicitly; starting the proxy never changes a database schema.
package migrations

import "embed"

// Files contains all Aegis up migrations compiled into aegisd.
//
//go:embed *.up.sql
var Files embed.FS
