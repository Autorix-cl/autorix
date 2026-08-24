import { describe, it, expect } from "vitest";
import { evalCel } from "./themis-evaluator";

describe("Themis CEL Evaluator", () => {
  it("should evaluate basic equality", () => {
    const context = { request: { role: "admin" } };
    expect(evalCel('request.role == "admin"', context)).toBe(true);
    expect(evalCel('request.role == "user"', context)).toBe(false);
  });

  it("should evaluate logical AND", () => {
    const context = { request: { role: "admin", action: "delete" } };
    expect(evalCel('request.role == "admin" && request.action == "delete"', context)).toBe(true);
    expect(evalCel('request.role == "admin" && request.action == "read"', context)).toBe(false);
  });

  it("should evaluate logical OR", () => {
    const context = { request: { role: "editor" } };
    expect(evalCel('request.role == "admin" || request.role == "editor"', context)).toBe(true);
  });

  it("should evaluate numeric comparisons", () => {
    const context = { resource: { size: 500 } };
    expect(evalCel('resource.size > 100', context)).toBe(true);
    expect(evalCel('resource.size < 1000', context)).toBe(true);
    expect(evalCel('resource.size >= 500', context)).toBe(true);
    expect(evalCel('resource.size <= 499', context)).toBe(false);
  });

  it("should handle deep nested properties", () => {
    const context = { request: { auth: { claims: { email: "alice@autorix.com" } } } };
    expect(evalCel('request.auth.claims.email == "alice@autorix.com"', context)).toBe(true);
  });

  it("should handle missing properties gracefully (return false or throw)", () => {
    const context = { request: {} };
    // Assuming we want undefined properties to evaluate safely to false rather than crashing
    expect(evalCel('request.auth.role == "admin"', context)).toBe(false);
  });

  it("should handle the NOT operator", () => {
    const context = { request: { is_banned: false } };
    expect(evalCel('!request.is_banned', context)).toBe(true);
  });

  it("should reject unsafe operations (prototype pollution or functions)", () => {
    const context = {};
    expect(() => evalCel('Math.random()', context)).toThrow();
    expect(() => evalCel('constructor.name', context)).toThrow();
  });
});
