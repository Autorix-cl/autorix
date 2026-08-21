package apl

import (
	"testing"

)

func TestParseNamespace(t *testing.T) {
	input := `
		import { Namespace, Context } from "@ory/keto-namespace-types"

		class Document implements Namespace {
			related: {
				owner: User[]
				editor: User[]
				viewer: User[]
				parent: Folder[]
			}

			permits = {
				view: (ctx: Context): boolean =>
					this.related.viewer.includes(ctx.subject) ||
					this.related.editor.includes(ctx.subject) ||
					this.related.owner.includes(ctx.subject) ||
					this.related.parent.traverse((p) => p.related.view),
				edit: (ctx: Context): boolean =>
					this.related.editor.includes(ctx.subject) ||
					this.related.owner,
				owner: (ctx: Context): boolean =>
					this.related.owner.includes(ctx.subject)
			}
		}
	`

	ns, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if ns.Name != "Document" {
		t.Errorf("Expected namespace name 'Document', got '%s'", ns.Name)
	}

	if len(ns.Relations) != 3 {
		t.Errorf("Expected 3 relations, got %d", len(ns.Relations))
	}

	ownerRel := ns.Relations["owner"]
	if ownerRel.Rewrite.Type != "computed_userset" || ownerRel.Rewrite.Relation != "owner" {
		t.Errorf("Expected owner relation to be computed_userset owner, got %+v", ownerRel.Rewrite)
	}

	editRel := ns.Relations["edit"]
	if editRel.Rewrite.Type != "union" {
		t.Errorf("Expected edit relation to be union, got %+v", editRel.Rewrite)
	}

	if len(editRel.Rewrite.Children) != 2 {
		t.Errorf("Expected edit relation to have 2 children, got %d", len(editRel.Rewrite.Children))
	}

	viewRel := ns.Relations["view"]
	if viewRel.Rewrite.Type != "union" {
		t.Errorf("Expected view relation to be union, got %+v", viewRel.Rewrite)
	}

	if len(viewRel.Rewrite.Children) != 4 {
		t.Errorf("Expected view relation to have 4 children, got %d", len(viewRel.Rewrite.Children))
	}
}

func TestSimpleParse(t *testing.T) {
	input := `class Document implements Namespace { related: { owner: User[] }; permits = { edit: (ctx) => this.related.owner } }`
	
	ns, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if ns.Name != "Document" {
		t.Errorf("Expected Document, got %s", ns.Name)
	}

	if len(ns.Relations) != 1 {
		t.Errorf("Expected 1 relation, got %d", len(ns.Relations))
	}

	editRel := ns.Relations["edit"]
	if editRel.Rewrite.Type != "computed_userset" || editRel.Rewrite.Relation != "owner" {
		t.Errorf("Expected edit relation to be computed_userset owner, got %+v", editRel.Rewrite)
	}
}
