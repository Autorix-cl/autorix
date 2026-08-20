package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/autorix/nexus/internal/core"
	"github.com/autorix/platform/paging"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Pool exposes the underlying connection pool so main.go can wire
// postgres.Check into the engine's health.Checker (ADR 0001).
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// WriteTuples inserts a batch of relation tuples transactionally
func (r *Repository) WriteTuples(ctx context.Context, tuples []core.Tuple) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once Commit has succeeded

	for _, t := range tuples {
		var caveatCtx []byte
		if t.CaveatContext != nil {
			var marshalErr error
			caveatCtx, marshalErr = json.Marshal(t.CaveatContext)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal caveat context: %w", marshalErr)
			}
		}

		var caveatName *string
		if t.CaveatName != "" {
			caveatName = &t.CaveatName
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO relation_tuples (
				namespace, object, relation, 
				subject_namespace, subject_object, subject_relation,
				caveat_name, caveat_context
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT DO NOTHING
		`, t.Namespace, t.Object, t.Relation,
			t.SubjectNamespace, t.SubjectObject, t.SubjectRelation,
			caveatName, caveatCtx)

		if err != nil {
			return fmt.Errorf("failed to insert tuple: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ReadTuples fetches tuples based on filters (namespace, object, relation)
func (r *Repository) ReadTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error) {
	query := `
		SELECT namespace, object, relation, 
		       subject_namespace, subject_object, subject_relation,
		       caveat_name, caveat_context
		FROM relation_tuples
		WHERE namespace = $1 AND object = $2 AND relation = $3
	`

	rows, err := r.pool.Query(ctx, query, filter.Namespace, filter.Object, filter.Relation)
	if err != nil {
		return nil, fmt.Errorf("failed to query tuples: %w", err)
	}
	defer rows.Close()

	var result []core.Tuple
	for rows.Next() {
		var t core.Tuple
		var caveatName *string
		var caveatCtx []byte

		err := rows.Scan(
			&t.Namespace, &t.Object, &t.Relation,
			&t.SubjectNamespace, &t.SubjectObject, &t.SubjectRelation,
			&caveatName, &caveatCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tuple: %w", err)
		}

		if caveatName != nil {
			t.CaveatName = *caveatName
		}
		if caveatCtx != nil {
			if unmarshalErr := json.Unmarshal(caveatCtx, &t.CaveatContext); unmarshalErr != nil {
				return nil, fmt.Errorf("failed to unmarshal caveat context: %w", unmarshalErr)
			}
		}
		result = append(result, t)
	}

	return result, nil
}

// tupleCursor is the JSON payload wrapped by paging.EncodeCursor for
// ListTuples. relation_tuples has no synthetic ID — its primary key is the
// six-column (namespace, object, relation, subject_*) tuple — so the cursor
// carries commit_time plus that full key for a strictly ordered,
// tie-break-safe keyset (commit_time alone is not unique: WriteTuples
// commits a whole batch within one transaction, and CURRENT_TIMESTAMP is
// constant for the duration of a transaction).
type tupleCursor struct {
	CommitTime       time.Time `json:"t"`
	Namespace        string    `json:"ns"`
	Object           string    `json:"o"`
	Relation         string    `json:"r"`
	SubjectNamespace string    `json:"sns"`
	SubjectObject    string    `json:"so"`
	SubjectRelation  string    `json:"sr"`
}

func decodeTupleCursor(cursor string) (tupleCursor, error) {
	decoded, err := paging.DecodeCursor(cursor)
	if err != nil {
		return tupleCursor{}, err
	}
	var c tupleCursor
	if err := json.Unmarshal([]byte(decoded), &c); err != nil {
		return tupleCursor{}, fmt.Errorf("malformed cursor payload: %w", err)
	}
	return c, nil
}

// ListTuples returns tuples for admin/console listing using real SQL keyset
// pagination, optionally filtered by namespace. An empty namespace lists
// tuples across all namespaces, most recent first. At most limit+1 rows are
// fetched to detect whether more exist beyond the page, and a non-empty
// cursor restricts the query to rows strictly after the last one returned
// by a previous call.
func (r *Repository) ListTuples(ctx context.Context, namespace string, limit int, cursor string) ([]core.Tuple, bool, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT namespace, object, relation,
		       subject_namespace, subject_object, subject_relation,
		       caveat_name, caveat_context, commit_time
		FROM relation_tuples
	`
	args := []interface{}{}
	var conditions []string
	if namespace != "" {
		args = append(args, namespace)
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", len(args)))
	}
	if cursor != "" {
		c, err := decodeTupleCursor(cursor)
		if err != nil {
			return nil, false, fmt.Errorf("invalid cursor: %w", err)
		}
		args = append(args, c.CommitTime, c.Namespace, c.Object, c.Relation, c.SubjectNamespace, c.SubjectObject, c.SubjectRelation)
		base := len(args) - 6
		conditions = append(conditions, fmt.Sprintf(
			"(commit_time, namespace, object, relation, subject_namespace, subject_object, subject_relation) < ($%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			base, base+1, base+2, base+3, base+4, base+5, base+6,
		))
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += fmt.Sprintf(
		" ORDER BY commit_time DESC, namespace DESC, object DESC, relation DESC, subject_namespace DESC, subject_object DESC, subject_relation DESC LIMIT $%d",
		len(args)+1,
	)
	args = append(args, limit+1)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list tuples: %w", err)
	}
	defer rows.Close()

	var result []core.Tuple
	for rows.Next() {
		var t core.Tuple
		var caveatName *string
		var caveatCtx []byte

		err := rows.Scan(
			&t.Namespace, &t.Object, &t.Relation,
			&t.SubjectNamespace, &t.SubjectObject, &t.SubjectRelation,
			&caveatName, &caveatCtx, &t.CommitTime,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan tuple: %w", err)
		}

		if caveatName != nil {
			t.CaveatName = *caveatName
		}
		if caveatCtx != nil {
			if unmarshalErr := json.Unmarshal(caveatCtx, &t.CaveatContext); unmarshalErr != nil {
				return nil, false, fmt.Errorf("failed to unmarshal caveat context: %w", unmarshalErr)
			}
		}
		result = append(result, t)
	}

	hasMore := len(result) > limit
	if hasMore {
		result = result[:limit]
	}
	return result, hasMore, nil
}

// DeleteTuples removes a batch of relation tuples transactionally, matched by
// their full primary key (namespace/object/relation/subject).
func (r *Repository) DeleteTuples(ctx context.Context, tuples []core.Tuple) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // no-op once Commit has succeeded

	for _, t := range tuples {
		_, err = tx.Exec(ctx, `
			DELETE FROM relation_tuples
			WHERE namespace = $1 AND object = $2 AND relation = $3
			  AND subject_namespace = $4 AND subject_object = $5 AND subject_relation = $6
		`, t.Namespace, t.Object, t.Relation,
			t.SubjectNamespace, t.SubjectObject, t.SubjectRelation)

		if err != nil {
			return fmt.Errorf("failed to delete tuple: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// QueryTuples fetches tuples matching any non-empty fields in the filter.
func (r *Repository) QueryTuples(ctx context.Context, filter core.Tuple) ([]core.Tuple, error) {
	query := `
		SELECT namespace, object, relation, 
		       subject_namespace, subject_object, subject_relation,
		       caveat_name, caveat_context
		FROM relation_tuples
	`
	var conditions []string
	var args []interface{}

	if filter.Namespace != "" {
		args = append(args, filter.Namespace)
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", len(args)))
	}
	if filter.Object != "" {
		args = append(args, filter.Object)
		conditions = append(conditions, fmt.Sprintf("object = $%d", len(args)))
	}
	if filter.Relation != "" {
		args = append(args, filter.Relation)
		conditions = append(conditions, fmt.Sprintf("relation = $%d", len(args)))
	}
	if filter.SubjectNamespace != "" {
		args = append(args, filter.SubjectNamespace)
		conditions = append(conditions, fmt.Sprintf("subject_namespace = $%d", len(args)))
	}
	if filter.SubjectObject != "" {
		args = append(args, filter.SubjectObject)
		conditions = append(conditions, fmt.Sprintf("subject_object = $%d", len(args)))
	}
	if filter.SubjectRelation != "" {
		args = append(args, filter.SubjectRelation)
		conditions = append(conditions, fmt.Sprintf("subject_relation = $%d", len(args)))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tuples: %w", err)
	}
	defer rows.Close()

	var result []core.Tuple
	for rows.Next() {
		var t core.Tuple
		var caveatName *string
		var caveatCtx []byte

		err := rows.Scan(
			&t.Namespace, &t.Object, &t.Relation,
			&t.SubjectNamespace, &t.SubjectObject, &t.SubjectRelation,
			&caveatName, &caveatCtx,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tuple: %w", err)
		}

		if caveatName != nil {
			t.CaveatName = *caveatName
		}
		if caveatCtx != nil {
			if unmarshalErr := json.Unmarshal(caveatCtx, &t.CaveatContext); unmarshalErr != nil {
				return nil, fmt.Errorf("failed to unmarshal caveat context: %w", unmarshalErr)
			}
		}
		result = append(result, t)
	}

	return result, nil
}

// WriteCaveat creates or updates a caveat definition.
func (r *Repository) WriteCaveat(ctx context.Context, caveat core.CaveatDefinition) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO caveats (name, cel_expression)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET cel_expression = EXCLUDED.cel_expression
	`, caveat.Name, caveat.CELExpression)
	if err != nil {
		return fmt.Errorf("failed to write caveat: %w", err)
	}
	return nil
}

// GetCaveat retrieves a caveat definition by name.
func (r *Repository) GetCaveat(ctx context.Context, name string) (*core.CaveatDefinition, error) {
	var c core.CaveatDefinition
	err := r.pool.QueryRow(ctx, `SELECT name, cel_expression, created_at FROM caveats WHERE name = $1`, name).
		Scan(&c.Name, &c.CELExpression, &c.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get caveat: %w", err)
	}
	return &c, nil
}

// GetCaveatExpression implements caveat.CaveatGetter.
func (r *Repository) GetCaveatExpression(ctx context.Context, name string) (string, error) {
	c, err := r.GetCaveat(ctx, name)
	if err != nil {
		return "", err
	}
	if c == nil {
		return "", fmt.Errorf("caveat %q not found", name)
	}
	return c.CELExpression, nil
}

// ListCaveats retrieves all caveats ordered by name.
func (r *Repository) ListCaveats(ctx context.Context) ([]core.CaveatDefinition, error) {
	rows, err := r.pool.Query(ctx, `SELECT name, cel_expression, created_at FROM caveats ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list caveats: %w", err)
	}
	defer rows.Close()

	var result []core.CaveatDefinition
	for rows.Next() {
		var c core.CaveatDefinition
		if err := rows.Scan(&c.Name, &c.CELExpression, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan caveat: %w", err)
		}
		result = append(result, c)
	}
	return result, nil
}

// DeleteCaveat removes a caveat definition by name.
func (r *Repository) DeleteCaveat(ctx context.Context, name string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM caveats WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete caveat: %w", err)
	}
	return nil
}

// WriteNamespace creates or updates a namespace schema.
func (r *Repository) WriteNamespace(ctx context.Context, schema core.NamespaceSchema) error {
	data, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal namespace schema: %w", err)
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO namespace_schemas (name, schema, updated_at)
		VALUES ($1, $2, CURRENT_TIMESTAMP)
		ON CONFLICT (name) DO UPDATE SET schema = EXCLUDED.schema, updated_at = CURRENT_TIMESTAMP
	`, schema.Name, data)
	if err != nil {
		return fmt.Errorf("failed to write namespace schema: %w", err)
	}
	return nil
}

// GetNamespace retrieves a namespace schema by name.
func (r *Repository) GetNamespace(ctx context.Context, name string) (*core.NamespaceSchema, error) {
	var raw []byte
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT schema, created_at, updated_at FROM namespace_schemas WHERE name = $1`, name).
		Scan(&raw, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get namespace schema: %w", err)
	}

	var schema core.NamespaceSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("failed to unmarshal namespace schema: %w", err)
	}
	schema.CreatedAt = createdAt
	schema.UpdatedAt = updatedAt
	return &schema, nil
}

// ListNamespaces lists all namespace schemas.
func (r *Repository) ListNamespaces(ctx context.Context) ([]core.NamespaceSchema, error) {
	rows, err := r.pool.Query(ctx, `SELECT schema, created_at, updated_at FROM namespace_schemas ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespace schemas: %w", err)
	}
	defer rows.Close()

	var result []core.NamespaceSchema
	for rows.Next() {
		var raw []byte
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&raw, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan namespace schema: %w", err)
		}
		var schema core.NamespaceSchema
		if err := json.Unmarshal(raw, &schema); err != nil {
			return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
		}
		schema.CreatedAt = createdAt
		schema.UpdatedAt = updatedAt
		result = append(result, schema)
	}
	return result, nil
}

// DeleteNamespace removes a namespace schema by name.
func (r *Repository) DeleteNamespace(ctx context.Context, name string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM namespace_schemas WHERE name = $1`, name)
	if err != nil {
		return fmt.Errorf("failed to delete namespace schema: %w", err)
	}
	return nil
}
