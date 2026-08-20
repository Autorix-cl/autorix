package core

import (
	"context"
	"time"
)

// Tuple represents a Zanzibar relationship tuple with ABAC integration
type Tuple struct {
	Namespace        string                 `json:"namespace"`
	Object           string                 `json:"object"`
	Relation         string                 `json:"relation"`
	SubjectNamespace string                 `json:"subject_namespace"`
	SubjectObject    string                 `json:"subject_object"`
	SubjectRelation  string                 `json:"subject_relation,omitempty"`

	CaveatName    string                 `json:"caveat_name,omitempty"`
	CaveatContext map[string]interface{} `json:"caveat_context,omitempty"`

	// CommitTime is populated by ListTuples for keyset pagination; it is the
	// zero value everywhere else (Check, ReadTuples, WriteTuples input, ...).
	CommitTime time.Time `json:"commit_time,omitempty"`
}

// CaveatDefinition represents an ABAC rule compiled in CEL
type CaveatDefinition struct {
	Name          string    `json:"name"`
	CELExpression string    `json:"cel_expression"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// CaveatResult records the evaluation of a single caveat during check
type CaveatResult struct {
	CaveatName string                 `json:"caveat_name"`
	Allowed    bool                   `json:"allowed"`
	Reason     string                 `json:"reason,omitempty"`
	Context    map[string]interface{} `json:"context,omitempty"`
}

// DecisionNode represents a node in the decision tree trace for explainability
type DecisionNode struct {
	NodeID      string          `json:"node_id,omitempty"`
	Namespace   string          `json:"namespace"`
	Object      string          `json:"object"`
	Relation    string          `json:"relation"`
	Subject     Tuple           `json:"subject"`
	Allowed     bool            `json:"allowed"`
	Reason      string          `json:"reason,omitempty"`
	Caveat      *CaveatResult   `json:"caveat,omitempty"`
	RewriteType string          `json:"rewrite_type,omitempty"`
	Children    []*DecisionNode `json:"children,omitempty"`
}

// RewriteRule defines rewrite operations on relations
type RewriteRule struct {
	Type             string         `json:"type"` // "this", "computed_userset", "tuple_to_userset", "union", "intersection"
	Relation         string         `json:"relation,omitempty"`          // for computed_userset
	TuplesetRelation string         `json:"tupleset_relation,omitempty"` // for tuple_to_userset
	ComputedRelation string         `json:"computed_relation,omitempty"` // for tuple_to_userset
	Children         []*RewriteRule `json:"children,omitempty"`          // for union / intersection
}

// RelationDefinition defines the rules for a relation in a namespace
type RelationDefinition struct {
	Rewrite *RewriteRule `json:"rewrite,omitempty"`
}

// NamespaceSchema models a namespace and its relation definitions/rewrites
type NamespaceSchema struct {
	Name      string                        `json:"name"`
	Relations map[string]RelationDefinition `json:"relations"`
	CreatedAt time.Time                     `json:"created_at,omitempty"`
	UpdatedAt time.Time                     `json:"updated_at,omitempty"`
}

// CheckRequest is the input to permission evaluation
type CheckRequest struct {
	Namespace string
	Object    string
	Relation  string
	Subject   Tuple // Simplified representation of the subject

	// RequestContext provided by the API gateway (Aegis)
	RequestContext map[string]interface{}

	// Explain requests a detailed decision tree trace
	Explain bool
}

// CheckResult contains whether access is allowed, a human-readable reason, and optional decision trace
type CheckResult struct {
	Allowed bool
	Reason  string
	Trace   *DecisionNode
}

// ExpandRequest asks for the full subtree of permissions for an object#relation
type ExpandRequest struct {
	Namespace string
	Object    string
	Relation  string
}

// ExpandTree represents the expanded relation tree
type ExpandTree struct {
	Type     string        `json:"type"` // "leaf", "union", "intersection", "userset"
	Tuple    *Tuple        `json:"tuple,omitempty"`
	Children []*ExpandTree `json:"children,omitempty"`
}

// LookupSubjectsRequest asks who has a relation on a specific resource
type LookupSubjectsRequest struct {
	Namespace string
	Object    string
	Relation  string
}

// LookupResourcesRequest asks what resources in a namespace a subject can access with a relation
type LookupResourcesRequest struct {
	Namespace        string
	Relation         string
	SubjectNamespace string
	SubjectObject    string
	SubjectRelation  string
}

// CaveatEvaluator handles compilation and execution of CEL expressions
type CaveatEvaluator interface {
	Compile(expression string) error
	Validate(expression string) error
	Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error)
	EvaluateByName(ctx context.Context, name string, reqCtx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error)
}

// GraphEngine is responsible for evaluating Hybrid ReBAC+ABAC permissions
type GraphEngine interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
	Expand(ctx context.Context, req ExpandRequest) (*ExpandTree, error)
	LookupSubjects(ctx context.Context, req LookupSubjectsRequest) ([]Tuple, error)
	LookupResources(ctx context.Context, req LookupResourcesRequest) ([]string, error)
}
