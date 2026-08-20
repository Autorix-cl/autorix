package paging_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autorix/platform/paging"
)

func TestParseRequest_DefaultsWhenNoParamsGiven(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/widgets", nil)

	got := paging.ParseRequest(req)

	if got.Cursor != "" {
		t.Errorf("expected empty cursor, got %q", got.Cursor)
	}
	if got.Limit != paging.DefaultLimit {
		t.Errorf("expected default limit %d, got %d", paging.DefaultLimit, got.Limit)
	}
	if got.Sort != "" {
		t.Errorf("expected empty sort, got %q", got.Sort)
	}
	if len(got.Filter) != 0 {
		t.Errorf("expected empty filter map, got %v", got.Filter)
	}
}

func TestParseRequest_ReadsCursorLimitAndSort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/widgets?cursor=abc123&limit=50&sort=-created_at", nil)

	got := paging.ParseRequest(req)

	if got.Cursor != "abc123" {
		t.Errorf("expected cursor abc123, got %q", got.Cursor)
	}
	if got.Limit != 50 {
		t.Errorf("expected limit 50, got %d", got.Limit)
	}
	if got.Sort != "-created_at" {
		t.Errorf("expected sort -created_at, got %q", got.Sort)
	}
}

func TestParseRequest_ClampsLimitToMax(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/widgets?limit=99999", nil)

	got := paging.ParseRequest(req)

	if got.Limit != paging.MaxLimit {
		t.Errorf("expected limit clamped to MaxLimit=%d, got %d", paging.MaxLimit, got.Limit)
	}
}

func TestParseRequest_FallsBackToDefaultOnNonPositiveLimit(t *testing.T) {
	for _, raw := range []string{"0", "-5", "not-a-number"} {
		req := httptest.NewRequest(http.MethodGet, "/widgets?limit="+raw, nil)
		got := paging.ParseRequest(req)
		if got.Limit != paging.DefaultLimit {
			t.Errorf("limit=%q: expected fallback to default %d, got %d", raw, paging.DefaultLimit, got.Limit)
		}
	}
}

func TestParseRequest_CollectsFilterParamsWithPrefix(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/widgets?filter.status=active&filter.status=pending&filter.owner=alice", nil)

	got := paging.ParseRequest(req)

	if len(got.Filter["status"]) != 2 || got.Filter["status"][0] != "active" || got.Filter["status"][1] != "pending" {
		t.Errorf("expected status filter [active pending], got %v", got.Filter["status"])
	}
	if len(got.Filter["owner"]) != 1 || got.Filter["owner"][0] != "alice" {
		t.Errorf("expected owner filter [alice], got %v", got.Filter["owner"])
	}
}

func TestCursor_EncodeDecodeRoundTrips(t *testing.T) {
	encoded := paging.EncodeCursor("id:12345")

	decoded, err := paging.DecodeCursor(encoded)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decoded != "id:12345" {
		t.Fatalf("expected round-trip to recover original value, got %q", decoded)
	}
}

func TestCursor_DecodeRejectsGarbage(t *testing.T) {
	_, err := paging.DecodeCursor("not-valid-base64!!!")
	if err == nil {
		t.Fatalf("expected an error decoding a garbage cursor")
	}
}

func TestWriteEnvelope_ProducesExpectedShape(t *testing.T) {
	rec := httptest.NewRecorder()

	paging.WriteEnvelope(rec, []string{"a", "b"}, "next-cursor-value", true)

	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}

	var body struct {
		Data       []string `json:"data"`
		NextCursor string   `json:"next_cursor"`
		HasMore    bool     `json:"has_more"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if len(body.Data) != 2 || body.Data[0] != "a" {
		t.Fatalf("expected data [a b], got %v", body.Data)
	}
	if body.NextCursor != "next-cursor-value" {
		t.Fatalf("expected next_cursor echoed, got %q", body.NextCursor)
	}
	if !body.HasMore {
		t.Fatalf("expected has_more=true")
	}
}

func TestWriteEnvelope_OmitsNextCursorWhenEmpty(t *testing.T) {
	rec := httptest.NewRecorder()

	paging.WriteEnvelope(rec, []string{}, "", false)

	var raw map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if _, present := raw["next_cursor"]; present {
		t.Fatalf("expected next_cursor to be omitted when empty, got %v", raw["next_cursor"])
	}
}
