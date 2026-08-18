package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/autorix/nexus/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// WriteTuples inserts a batch of relation tuples transactionally
func (r *Repository) WriteTuples(ctx context.Context, tuples []core.Tuple) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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

// ListTuples returns tuples for admin/console listing, optionally filtered by namespace.
// An empty namespace lists tuples across all namespaces, most recent first.
func (r *Repository) ListTuples(ctx context.Context, namespace string, limit int) ([]core.Tuple, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT namespace, object, relation,
		       subject_namespace, subject_object, subject_relation,
		       caveat_name, caveat_context
		FROM relation_tuples
	`
	args := []interface{}{}
	if namespace != "" {
		query += " WHERE namespace = $1"
		args = append(args, namespace)
	}
	query += fmt.Sprintf(" ORDER BY commit_time DESC LIMIT $%d", len(args)+1)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tuples: %w", err)
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

// DeleteTuples removes a batch of relation tuples transactionally, matched by
// their full primary key (namespace/object/relation/subject).
func (r *Repository) DeleteTuples(ctx context.Context, tuples []core.Tuple) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

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
