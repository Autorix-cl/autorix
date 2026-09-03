import { describe, expect, it } from "vitest";
import {
  checkResponseSchema,

  tupleListSchema,
  tupleSchema,
  expandResponseSchema,
  lookupSubjectsResponseSchema,
  lookupResourcesResponseSchema,
  namespaceSchema,
  caveatSchema,
} from "./nexus";


describe("tupleSchema", () => {
  it("accepts a realistic tuple payload", () => {
    const payload = {
      namespace: "document",
      object: "doc:42",
      relation: "viewer",
      subject_namespace: "user",
      subject_id: "user:alice",
      subject_relation: "",
      caveat_name: "expires_at",
      caveat_context: { expires_at: "2026-01-01T00:00:00Z" },
    };
    expect(tupleSchema.safeParse(payload).success).toBe(true);
  });

  it("accepts a list response", () => {
    expect(
      tupleListSchema.safeParse([
        {
          namespace: "document",
          object: "doc:42",
          relation: "viewer",
          subject_namespace: "user",
          subject_id: "user:alice",
        },
      ]).success,
    ).toBe(true);
  });

  it("rejects a tuple missing required fields", () => {
    const payload = { namespace: "document", object: "doc:42" };
    expect(tupleSchema.safeParse(payload).success).toBe(false);
  });
});

describe("checkResponseSchema", () => {
  it("accepts a valid check response without trace", () => {
    expect(checkResponseSchema.safeParse({ allowed: true, reason: "matched" }).success).toBe(true);
  });

  it("accepts a check response with recursive decision trace and caveat", () => {
    const payload = {
      allowed: true,
      reason: "direct relation matched with caveat",
      trace: {
        node_id: "node_1",
        namespace: "document",
        object: "doc:123",
        relation: "viewer",
        subject: {
          namespace: "user",
          object: "alice",
          relation: "",
        },
        allowed: true,
        reason: "caveat condition evaluated to true",
        caveat: {
          caveat_name: "time_window",
          allowed: true,
          reason: "within business hours",
          context: { current_hour: 14 },
        },
        rewrite_type: "this",
        children: [
          {
            node_id: "node_child_1",
            namespace: "document",
            object: "doc:123",
            relation: "parent_viewer",
            allowed: true,
          },
        ],
      },
    };
    expect(checkResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("rejects a response missing allowed", () => {
    expect(checkResponseSchema.safeParse({ reason: "matched" }).success).toBe(false);
  });
});

describe("expandResponseSchema", () => {
  it("validates recursive relation expansion tree", () => {
    const payload = {
      tree: {
        type: "union",
        children: [
          {
            type: "leaf",
            tuple: {
              namespace: "document",
              object: "doc:1",
              relation: "viewer",
              subject_namespace: "user",
              subject_id: "alice",
            },
          },
          {
            type: "userset",
            children: [
              {
                type: "leaf",
                tuple: {
                  namespace: "organization",
                  object: "org:1",
                  relation: "member",
                  subject_namespace: "user",
                  subject_id: "bob",
                },
              },
            ],
          },
        ],
      },
    };
    expect(expandResponseSchema.safeParse(payload).success).toBe(true);
  });
});

describe("lookup schemas", () => {
  it("validates lookupSubjectsResponseSchema", () => {
    const payload = {
      subjects: [
        { namespace: "user", object: "alice" },
        { namespace: "user", object: "bob", relation: "member" },
      ],
    };
    expect(lookupSubjectsResponseSchema.safeParse(payload).success).toBe(true);
  });

  it("validates lookupResourcesResponseSchema", () => {
    const payload = {
      resources: ["doc:1", "doc:2", "doc:99"],
    };
    expect(lookupResourcesResponseSchema.safeParse(payload).success).toBe(true);
  });
});

describe("namespaceSchema & caveatSchema", () => {
  it("validates namespace schema with relations and rewrites", () => {
    const payload = {
      name: "document",
      relations: {
        viewer: {
          rewrite: {
            type: "union",
            children: [
              { type: "this" },
              { type: "computed_userset", relation: "editor" },
            ],
          },
        },
      },
      created_at: "2026-09-02T20:00:00Z",
    };
    const parsed = namespaceSchema.safeParse(payload);
    expect(parsed.success).toBe(true);
  });

  it("validates caveat definition", () => {
    const payload = {
      name: "is_business_hours",
      cel_expression: "request.time.hour >= 9 && request.time.hour < 18",
    };
    expect(caveatSchema.safeParse(payload).success).toBe(true);
  });
});

