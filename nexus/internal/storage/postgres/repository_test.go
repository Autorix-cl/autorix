package postgres_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/autorix/nexus/internal/core"
	"github.com/autorix/nexus/internal/storage/postgres"
	"github.com/autorix/platform/paging"
	"github.com/autorix/platform/pgtest"
)

func newTestRepo(t *testing.T) *postgres.Repository {
	t.Helper()
	pool := pgtest.StartPostgres(t, "../../../migrations")
	return postgres.NewRepository(pool)
}

func TestWriteAndReadTuples(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuples := []core.Tuple{
		{
			Namespace: "document", Object: "doc1", Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "alice", SubjectRelation: "",
		},
		{
			Namespace: "document", Object: "doc1", Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "bob", SubjectRelation: "",
		},
	}

	if err := repo.WriteTuples(ctx, tuples); err != nil {
		t.Fatalf("WriteTuples: %v", err)
	}

	got, err := repo.ReadTuples(ctx, core.Tuple{Namespace: "document", Object: "doc1", Relation: "viewer"})
	if err != nil {
		t.Fatalf("ReadTuples: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tuples, got %d: %+v", len(got), got)
	}
}

func TestWriteTuplesIdempotentOnConflict(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuple := core.Tuple{
		Namespace: "document", Object: "doc1", Relation: "viewer",
		SubjectNamespace: "user", SubjectObject: "alice", SubjectRelation: "",
	}

	if err := repo.WriteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("first WriteTuples: %v", err)
	}
	// Writing the exact same tuple again should be a no-op (ON CONFLICT DO NOTHING),
	// not an error, thanks to the composite primary key.
	if err := repo.WriteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("second WriteTuples (duplicate) should not error: %v", err)
	}

	got, err := repo.ReadTuples(ctx, core.Tuple{Namespace: "document", Object: "doc1", Relation: "viewer"})
	if err != nil {
		t.Fatalf("ReadTuples: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected exactly 1 tuple after duplicate write, got %d", len(got))
	}
}

// TestWriteTuplesRollsBackOnMidBatchFailure confirms atomicity of WriteTuples:
// a batch where an early row is valid and a later row violates the
// caveat_name -> caveats(name) foreign key must leave NO tuples committed,
// including the valid one that preceded the bad row.
func TestWriteTuplesRollsBackOnMidBatchFailure(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuples := []core.Tuple{
		{
			// Valid tuple, would succeed on its own.
			Namespace: "document", Object: "doc-partial", Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "alice", SubjectRelation: "",
		},
		{
			// Invalid: caveat_name references a caveat that doesn't exist,
			// violating the FK constraint and aborting the whole tx.
			Namespace: "document", Object: "doc-partial", Relation: "editor",
			SubjectNamespace: "user", SubjectObject: "bob", SubjectRelation: "",
			CaveatName: "nonexistent-caveat",
		},
	}

	if err := repo.WriteTuples(ctx, tuples); err == nil {
		t.Fatalf("expected WriteTuples to fail due to caveat FK violation")
	}

	got, err := repo.ReadTuples(ctx, core.Tuple{Namespace: "document", Object: "doc-partial", Relation: "viewer"})
	if err != nil {
		t.Fatalf("ReadTuples: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no tuples committed after rolled-back batch, got %d: %+v", len(got), got)
	}
}

func TestDeleteTuples(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuple := core.Tuple{
		Namespace: "document", Object: "doc2", Relation: "viewer",
		SubjectNamespace: "user", SubjectObject: "carol", SubjectRelation: "",
	}
	if err := repo.WriteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("WriteTuples: %v", err)
	}

	if err := repo.DeleteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("DeleteTuples: %v", err)
	}

	got, err := repo.ReadTuples(ctx, core.Tuple{Namespace: "document", Object: "doc2", Relation: "viewer"})
	if err != nil {
		t.Fatalf("ReadTuples: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected tuple to be deleted, got %d: %+v", len(got), got)
	}
}

func TestDeleteTuplesNonExistentIsNoop(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuple := core.Tuple{
		Namespace: "document", Object: "doc-none", Relation: "viewer",
		SubjectNamespace: "user", SubjectObject: "nobody", SubjectRelation: "",
	}

	// Deleting a tuple that was never written should not error.
	if err := repo.DeleteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("DeleteTuples on nonexistent tuple: %v", err)
	}
}

func TestListTuplesFilterAndLimit(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuples := []core.Tuple{
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc2", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "bob"},
		{Namespace: "folder", Object: "f1", Relation: "owner", SubjectNamespace: "user", SubjectObject: "carol"},
	}
	if err := repo.WriteTuples(ctx, tuples); err != nil {
		t.Fatalf("WriteTuples: %v", err)
	}

	all, hasMore, err := repo.ListTuples(ctx, "", 100, "")
	if err != nil {
		t.Fatalf("ListTuples(all): %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 tuples across all namespaces, got %d", len(all))
	}
	if hasMore {
		t.Fatalf("expected hasMore=false when everything fits in one page")
	}

	docsOnly, _, err := repo.ListTuples(ctx, "document", 100, "")
	if err != nil {
		t.Fatalf("ListTuples(document): %v", err)
	}
	if len(docsOnly) != 2 {
		t.Fatalf("expected 2 document tuples, got %d", len(docsOnly))
	}

	limited, hasMoreLimited, err := repo.ListTuples(ctx, "", 1, "")
	if err != nil {
		t.Fatalf("ListTuples(limit=1): %v", err)
	}
	if len(limited) != 1 {
		t.Fatalf("expected limit=1 to return exactly 1 tuple, got %d", len(limited))
	}
	if !hasMoreLimited {
		t.Fatalf("expected hasMore=true when limit=1 and 3 tuples exist")
	}
}

// TestListTuples_CursorPagination proves ListTuples paginates via a real
// SQL keyset: a page smaller than the full set returns exactly N rows plus
// a cursor, and following that cursor returns the rest with no duplicate
// and no skipped row. Each tuple is written in its own transaction so
// commit_time strictly increases across them.
func TestListTuples_CursorPagination(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	const total = 5
	keys := make(map[string]bool, total)
	for i := 0; i < total; i++ {
		obj := fmt.Sprintf("doc-cursor-%d", i)
		tuple := core.Tuple{
			Namespace: "document", Object: obj, Relation: "viewer",
			SubjectNamespace: "user", SubjectObject: "alice",
		}
		if err := repo.WriteTuples(ctx, []core.Tuple{tuple}); err != nil {
			t.Fatalf("WriteTuples(%s): %v", obj, err)
		}
		keys[obj] = true
		time.Sleep(2 * time.Millisecond)
	}

	const pageSize = 2
	var all []core.Tuple
	cursor := ""
	for i := 0; i < 10; i++ { // bounded loop guard against an infinite pagination bug
		page, hasMore, err := repo.ListTuples(ctx, "", pageSize, cursor)
		if err != nil {
			t.Fatalf("ListTuples(cursor=%q): %v", cursor, err)
		}
		all = append(all, page...)
		if !hasMore {
			break
		}
		if len(page) == 0 {
			t.Fatalf("hasMore=true but page was empty")
		}
		last := page[len(page)-1]
		payload, err := json.Marshal(struct {
			CommitTime       time.Time `json:"t"`
			Namespace        string    `json:"ns"`
			Object           string    `json:"o"`
			Relation         string    `json:"r"`
			SubjectNamespace string    `json:"sns"`
			SubjectObject    string    `json:"so"`
			SubjectRelation  string    `json:"sr"`
		}{last.CommitTime, last.Namespace, last.Object, last.Relation, last.SubjectNamespace, last.SubjectObject, last.SubjectRelation})
		if err != nil {
			t.Fatalf("marshal cursor payload: %v", err)
		}
		cursor = paging.EncodeCursor(string(payload))
	}

	if len(all) != total {
		t.Fatalf("expected %d total tuples across pages, got %d", total, len(all))
	}
	seen := map[string]bool{}
	for _, tup := range all {
		if seen[tup.Object] {
			t.Fatalf("duplicate tuple for object %q across pages", tup.Object)
		}
		seen[tup.Object] = true
		delete(keys, tup.Object)
	}
	if len(keys) != 0 {
		t.Fatalf("some tuples were skipped across pages: %v", keys)
	}
}

func TestWriteTuplesWithValidCaveat(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	pool := repo.Pool()
	if _, err := pool.Exec(ctx, `INSERT INTO caveats (name, cel_expression) VALUES ($1, $2)`,
		"business-hours", `request.time.getHours() >= 9 && request.time.getHours() < 17`); err != nil {
		t.Fatalf("seeding caveat: %v", err)
	}

	tuple := core.Tuple{
		Namespace: "document", Object: "doc-caveat", Relation: "viewer",
		SubjectNamespace: "user", SubjectObject: "dave", SubjectRelation: "",
		CaveatName:    "business-hours",
		CaveatContext: map[string]interface{}{"region": "us-east"},
	}

	if err := repo.WriteTuples(ctx, []core.Tuple{tuple}); err != nil {
		t.Fatalf("WriteTuples with valid caveat: %v", err)
	}

	got, err := repo.ReadTuples(ctx, core.Tuple{Namespace: "document", Object: "doc-caveat", Relation: "viewer"})
	if err != nil {
		t.Fatalf("ReadTuples: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 tuple, got %d", len(got))
	}
	if got[0].CaveatName != "business-hours" {
		t.Fatalf("expected caveat name to round-trip, got %q", got[0].CaveatName)
	}
	if got[0].CaveatContext["region"] != "us-east" {
		t.Fatalf("expected caveat context to round-trip, got %+v", got[0].CaveatContext)
	}
}

func TestCaveatsCRUD(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	c := core.CaveatDefinition{
		Name:          "ip_check",
		CELExpression: `ctx.ip == "10.0.0.1"`,
	}

	// 1. Write
	if err := repo.WriteCaveat(ctx, c); err != nil {
		t.Fatalf("WriteCaveat: %v", err)
	}

	// 2. Get
	got, err := repo.GetCaveat(ctx, "ip_check")
	if err != nil {
		t.Fatalf("GetCaveat: %v", err)
	}
	if got == nil || got.CELExpression != c.CELExpression {
		t.Fatalf("expected expression %q, got %+v", c.CELExpression, got)
	}

	// 3. GetExpression
	expr, err := repo.GetCaveatExpression(ctx, "ip_check")
	if err != nil || expr != c.CELExpression {
		t.Fatalf("GetCaveatExpression: expr=%q, err=%v", expr, err)
	}

	// 4. List
	list, err := repo.ListCaveats(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListCaveats: got %d, err=%v", len(list), err)
	}

	// 5. Delete
	if err := repo.DeleteCaveat(ctx, "ip_check"); err != nil {
		t.Fatalf("DeleteCaveat: %v", err)
	}

	gotAfter, err := repo.GetCaveat(ctx, "ip_check")
	if err == nil && gotAfter != nil {
		t.Fatalf("expected caveat to be deleted, got %+v", gotAfter)
	}
}

func TestNamespacesCRUD(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	schema := core.NamespaceSchema{
		Name: "document",
		Relations: map[string]core.RelationDefinition{
			"viewer": {
				Rewrite: &core.RewriteRule{
					Type: "union",
					Children: []*core.RewriteRule{
						{Type: "this"},
						{Type: "computed_userset", Relation: "editor"},
					},
				},
			},
			"editor": {
				Rewrite: &core.RewriteRule{
					Type: "this",
				},
			},
		},
	}

	// 1. Write
	if err := repo.WriteNamespace(ctx, schema); err != nil {
		t.Fatalf("WriteNamespace: %v", err)
	}

	// 2. Get
	got, err := repo.GetNamespace(ctx, "document")
	if err != nil {
		t.Fatalf("GetNamespace: %v", err)
	}
	if got == nil || got.Name != "document" || len(got.Relations) != 2 {
		t.Fatalf("unexpected namespace got: %+v", got)
	}

	// 3. List
	list, err := repo.ListNamespaces(ctx)
	if err != nil || len(list) != 1 {
		t.Fatalf("ListNamespaces: got %d, err=%v", len(list), err)
	}

	// 4. Delete
	if err := repo.DeleteNamespace(ctx, "document"); err != nil {
		t.Fatalf("DeleteNamespace: %v", err)
	}

	gotAfter, err := repo.GetNamespace(ctx, "document")
	if err == nil && gotAfter != nil {
		t.Fatalf("expected namespace to be deleted, got %+v", gotAfter)
	}
}

func TestQueryTuples(t *testing.T) {
	ctx := context.Background()
	repo := newTestRepo(t)

	tuples := []core.Tuple{
		{Namespace: "document", Object: "doc1", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc2", Relation: "viewer", SubjectNamespace: "user", SubjectObject: "alice"},
		{Namespace: "document", Object: "doc1", Relation: "editor", SubjectNamespace: "user", SubjectObject: "bob"},
		{Namespace: "group", Object: "eng", Relation: "member", SubjectNamespace: "user", SubjectObject: "alice"},
	}
	if err := repo.WriteTuples(ctx, tuples); err != nil {
		t.Fatalf("WriteTuples: %v", err)
	}

	// Query by subject
	bySubj, err := repo.QueryTuples(ctx, core.Tuple{SubjectNamespace: "user", SubjectObject: "alice"})
	if err != nil {
		t.Fatalf("QueryTuples by subject: %v", err)
	}
	if len(bySubj) != 3 {
		t.Fatalf("expected 3 tuples for user:alice, got %d", len(bySubj))
	}

	// Query by namespace and relation
	byRel, err := repo.QueryTuples(ctx, core.Tuple{Namespace: "document", Relation: "viewer"})
	if err != nil {
		t.Fatalf("QueryTuples by relation: %v", err)
	}
	if len(byRel) != 2 {
		t.Fatalf("expected 2 tuples for document#viewer, got %d", len(byRel))
	}
}
