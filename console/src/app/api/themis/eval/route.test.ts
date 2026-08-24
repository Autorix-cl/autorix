import { describe, it, expect, beforeEach, vi } from "vitest";
import { POST } from "./route";

vi.mock("@/lib/server/themis-db", () => {
  let store: any[] = [];
  return {
    themisDb: {
      getPolicies: () => store.sort((a, b) => a.priority - b.priority),
      addPolicy: (p: any) => store.push(p),
      _reset: () => { store = []; }
    }
  };
});

describe("Themis Eval API", () => {
  beforeEach(async () => {
    const { themisDb } = await import("@/lib/server/themis-db");
    (themisDb as any)._reset();
    
    // Add some mock policies
    themisDb.addPolicy({
      id: "p1", name: "Admin Only", expression: 'request.role == "admin"', priority: 10, enabled: true
    });
    themisDb.addPolicy({
      id: "p2", name: "User Read", expression: 'request.role == "user" && request.action == "read"', priority: 20, enabled: true
    });
    themisDb.addPolicy({
      id: "p3", name: "Disabled Policy", expression: 'request.role == "guest"', priority: 5, enabled: false
    });
  });

  it("should evaluate to Passed if a policy matches", async () => {
    const req = new Request("http://localhost/api/themis/eval", {
      method: "POST",
      body: JSON.stringify({
        context: { request: { role: "admin", action: "write" } }
      })
    });
    
    const res = await POST(req);
    expect(res.status).toBe(200);
    const body = await res.json();
    
    expect(body.result).toBe("Passed");
    expect(body.matchedPolicy.id).toBe("p1");
  });

  it("should evaluate to Failed if no policy matches", async () => {
    const req = new Request("http://localhost/api/themis/eval", {
      method: "POST",
      body: JSON.stringify({
        context: { request: { role: "guest" } } // matches p3 but p3 is disabled
      })
    });
    
    const res = await POST(req);
    expect(res.status).toBe(200);
    const body = await res.json();
    
    expect(body.result).toBe("Failed");
    expect(body.matchedPolicy).toBeNull();
  });

  it("should support evaluating a specific expression ad-hoc (dry run)", async () => {
    const req = new Request("http://localhost/api/themis/eval", {
      method: "POST",
      body: JSON.stringify({
        context: { request: { temp: true } },
        expression: 'request.temp == true'
      })
    });
    
    const res = await POST(req);
    expect(res.status).toBe(200);
    const body = await res.json();
    
    expect(body.result).toBe("Passed");
    expect(body.matchedPolicy).toBeNull();
  });
});
