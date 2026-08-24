/* eslint-disable */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { GET, POST } from "./route";

// We'll mock a db module similar to what we did for Vulcan
vi.mock("@/lib/server/themis-db", () => {
  let store: any[] = [];
  return {
    themisDb: {
      getPolicies: () => store.sort((a, b) => a.priority - b.priority),
      getPolicyById: (id: string) => store.find(p => p.id === id),
      addPolicy: (p: any) => store.push(p),
      togglePolicy: (id: string) => {
        const p = store.find(p => p.id === id);
        if (p) { p.enabled = !p.enabled; return true; }
        return false;
      },
      deletePolicy: (id: string) => {
        const initialLength = store.length;
        store = store.filter(p => p.id !== id);
        return store.length < initialLength;
      },
      _reset: () => { store = []; }
    }
  };
});

describe("Themis Policies API", () => {
  beforeEach(async () => {
    const { themisDb } = await import("@/lib/server/themis-db");
    (themisDb as any)._reset();
  });

  it("POST /api/themis/policies should create a valid policy", async () => {
    const payload = {
      name: "Only Admins",
      expression: 'request.role == "admin"',
      priority: 10,
    };
    
    const req = new Request("http://localhost/api/themis/policies", {
      method: "POST",
      body: JSON.stringify(payload)
    });
    
    const res = await POST(req);
    expect(res.status).toBe(201);
    
    const body = await res.json();
    expect(body.id).toBeDefined();
    expect(body.name).toBe("Only Admins");
    expect(body.enabled).toBe(true); // default true
  });

  it("POST /api/themis/policies should reject invalid CEL syntax", async () => {
    const payload = {
      name: "Bad Policy",
      expression: 'request.role =="', // syntax error
      priority: 10,
    };
    
    const req = new Request("http://localhost/api/themis/policies", {
      method: "POST",
      body: JSON.stringify(payload)
    });
    
    const res = await POST(req);
    expect(res.status).toBe(400);
    const body = await res.json();
    expect(body.error).toContain("Invalid CEL syntax");
  });

  it("GET /api/themis/policies should return policies sorted by priority", async () => {
    await POST(new Request("http://localhost", { method: "POST", body: JSON.stringify({ name: "P2", expression: "true", priority: 20 }) }));
    await POST(new Request("http://localhost", { method: "POST", body: JSON.stringify({ name: "P1", expression: "true", priority: 10 }) }));
    
    const req = new Request("http://localhost/api/themis/policies", { method: "GET" });
    const res = await GET();
    
    expect(res.status).toBe(200);
    const policies = await res.json();
    expect(policies).toHaveLength(2);
    // P1 has lower priority number, so it should come first
    expect(policies[0].name).toBe("P1");
    expect(policies[1].name).toBe("P2");
  });
});
