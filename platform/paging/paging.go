// Package paging defines Autorix's one cursor-pagination, sorting and
// filtering convention (P1-S2-T6 / originally P1-S2-T6, referenced by
// P1-S1-T2), with request parsing and a response envelope helper. Defined
// once here so every engine's list endpoints and the console's tables agree
// on the same shape — retrofitting a convention across seven engines later
// is far more expensive than establishing it once.
package paging

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const (
	// DefaultLimit is used when a request omits limit, or supplies a
	// non-positive or unparsable one.
	DefaultLimit = 20
	// MaxLimit bounds the largest page a client may request.
	MaxLimit = 100
)

// filterPrefix is the query-parameter prefix that marks a filter, e.g.
// ?filter.status=active.
const filterPrefix = "filter."

// Request is one parsed list request: an opaque cursor, a bounded page
// size, an optional sort key ("field" ascending, "-field" descending), and
// arbitrary named filters with multi-value support.
type Request struct {
	Cursor string
	Limit  int
	Sort   string
	Filter map[string][]string
}

// ParseRequest reads Request out of r's query string. An invalid or
// out-of-range limit falls back to DefaultLimit (clamped to MaxLimit on the
// high side) rather than failing the request — pagination bounds are a
// server-side guarantee, not something a malformed client input should be
// able to break.
func ParseRequest(r *http.Request) Request {
	q := r.URL.Query()

	limit := DefaultLimit
	if raw := q.Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	filter := make(map[string][]string)
	for key, values := range q {
		if name, ok := strings.CutPrefix(key, filterPrefix); ok {
			filter[name] = values
		}
	}

	return Request{
		Cursor: q.Get("cursor"),
		Limit:  limit,
		Sort:   q.Get("sort"),
		Filter: filter,
	}
}

// EncodeCursor produces an opaque, URL-safe cursor token for v. Callers
// should treat cursors as opaque; the encoding is not a stability contract
// across Autorix versions.
func EncodeCursor(v string) string {
	return base64.URLEncoding.EncodeToString([]byte(v))
}

// DecodeCursor reverses EncodeCursor, returning an error for a token that
// is not valid base64 (e.g. one a caller hand-edited or truncated).
func DecodeCursor(cursor string) (string, error) {
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Envelope is the uniform shape of a paginated list response.
type Envelope struct {
	Data       interface{} `json:"data"`
	NextCursor string      `json:"next_cursor,omitempty"`
	HasMore    bool        `json:"has_more"`
}

// WriteEnvelope writes data as a paginated Envelope with status 200. Pass
// an empty nextCursor when hasMore is false; it is omitted from the body.
func WriteEnvelope(w http.ResponseWriter, data interface{}, nextCursor string, hasMore bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(Envelope{
		Data:       data,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	})
}
