import { describe, it, expect } from "vitest";
import { z } from "zod";
import { validateResourceForm } from "./form-helpers";

const clientFormSchema = z.object({
  name: z.string().min(2, "Name must be at least 2 characters"),
  email: z.string().email("Invalid email address"),
  age: z.number().min(18, "Must be 18+").optional(),
});

describe("Resource Form Helpers (P4-S4-T1, P4-S4-T4)", () => {
  it("validates form data and returns field errors accurately", () => {
    const result = validateResourceForm(clientFormSchema, {
      name: "A",
      email: "not-an-email",
    });

    expect(result.isValid).toBe(false);
    expect(result.errors.name).toBe("Name must be at least 2 characters");
    expect(result.errors.email).toBe("Invalid email address");
  });

  it("passes validation on valid data payload", () => {
    const result = validateResourceForm(clientFormSchema, {
      name: "Alice Smith",
      email: "alice@example.com",
      age: 25,
    });

    expect(result.isValid).toBe(true);
    expect(result.errors).toEqual({});
    expect(result.data).toEqual({
      name: "Alice Smith",
      email: "alice@example.com",
      age: 25,
    });
  });
});
