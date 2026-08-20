package rule

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/autorix/platform/paging"
)

// newTestStore writes n minimal, valid rules to a temp YAML file (ids
// rule-0..rule-N-1, in that order) and loads them through NewStore, exactly
// as the real admin API does.
func newTestStore(t *testing.T, n int) *FileStore {
	t.Helper()

	rules := ""
	for i := 0; i < n; i++ {
		rules += fmt.Sprintf(`- id: "rule-%d"
  match:
    url: "/api/rule-%d"
    methods: ["GET"]
  upstream:
    url: "http://backend.local"
`, i, i)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "rules.yaml")
	if err := os.WriteFile(path, []byte(rules), 0o600); err != nil {
		t.Fatalf("failed to write test rules file: %v", err)
	}

	store, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: unexpected error: %v", err)
	}
	return store
}

func TestListPage_NoCursorReturnsFirstPage(t *testing.T) {
	store := newTestStore(t, 3)

	page, hasMore, err := store.ListPage(2, "")
	if err != nil {
		t.Fatalf("ListPage: unexpected error: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(page))
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true")
	}
	if page[0].ID != "rule-0" || page[1].ID != "rule-1" {
		t.Fatalf("unexpected page contents: %+v", page)
	}
}

func TestListPage_EmptyCursorDefaultsSensibly(t *testing.T) {
	store := newTestStore(t, 2)

	page, hasMore, err := store.ListPage(20, "")
	if err != nil {
		t.Fatalf("ListPage: unexpected error: %v", err)
	}
	if len(page) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(page))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when everything fits in one page")
	}
}

// TestListPage_CursorPagination proves a page smaller than the full set
// returns exactly N rules plus a cursor, and following that cursor returns
// the rest with no duplicate and no skipped rule.
func TestListPage_CursorPagination(t *testing.T) {
	const total = 5
	store := newTestStore(t, total)

	const pageSize = 2
	var all []string
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := store.ListPage(pageSize, cursor)
		if err != nil {
			t.Fatalf("ListPage(cursor=%q): unexpected error: %v", cursor, err)
		}
		for _, r := range page {
			all = append(all, r.ID)
		}
		if !hasMore {
			break
		}
		if len(page) == 0 {
			t.Fatalf("hasMore=true but page was empty")
		}
		cursor = paging.EncodeCursor(page[len(page)-1].ID)
	}

	if len(all) != total {
		t.Fatalf("expected %d total rules across pages, got %d: %v", total, len(all), all)
	}
	seen := map[string]bool{}
	for i, id := range all {
		if seen[id] {
			t.Fatalf("duplicate rule %q across pages", id)
		}
		seen[id] = true
		want := fmt.Sprintf("rule-%d", i)
		if id != want {
			t.Fatalf("page order broken or rule skipped: position %d = %q, want %q", i, id, want)
		}
	}
}

func TestListPage_UnknownCursorErrors(t *testing.T) {
	store := newTestStore(t, 2)

	_, _, err := store.ListPage(10, paging.EncodeCursor("does-not-exist"))
	if err == nil {
		t.Fatalf("expected an error for an unknown cursor rule id")
	}
}
