package core

import "context"

// Tuple represents a Zanzibar relationship tuple with ABAC integration
type Tuple struct {
	Namespace       string
	Object          string
	Relation        string
	SubjectNamespace string
	SubjectObject    string
	SubjectRelation  string
	
	CaveatName    string
	CaveatContext map[string]interface{}
}

// CaveatDefinition represents an ABAC rule compiled in CEL
type CaveatDefinition struct {
	Name          string
	CELExpression string
}

// CaveatEvaluator handles compilation and execution of CEL expressions
type CaveatEvaluator interface {
	Compile(expression string) error
	Evaluate(ctx map[string]interface{}, caveatCtx map[string]interface{}) (bool, error)
}

// GraphEngine is responsible for evaluating Hybrid ReBAC+ABAC permissions
type GraphEngine interface {
	Check(ctx context.Context, req CheckRequest) (CheckResult, error)
	Expand(ctx context.Context, req ExpandRequest) (*ExpandTree, error)
}

type CheckRequest struct {
	Namespace string
	Object    string
	Relation  string
	Subject   Tuple // Simplified representation of the subject
	
	// RequestContext provided by the API gateway (Aegis)
	RequestContext map[string]interface{}
}

type CheckResult struct {
	Allowed bool
	Reason  string
}

type ExpandRequest struct {
	Namespace string
	Object    string
	Relation  string
}

type ExpandTree struct {
	// Representation of the expanded tree
}
