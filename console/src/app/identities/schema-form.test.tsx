import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { DynamicSchemaForm } from "./schema-form";

const mockSchema = {
  title: "User Identity",
  type: "object",
  properties: {
    traits: {
      type: "object",
      properties: {
        email: {
          type: "string",
          format: "email",
          title: "E-Mail Address",
          "autorix.io/credentials": {
            password: { identifier: true },
          },
        },
        name: {
          type: "object",
          title: "Full Name",
          properties: {
            first: { type: "string", title: "First Name" },
            last: { type: "string", title: "Last Name" },
          },
          required: ["first"],
        },
        marketing_opt_in: {
          type: "boolean",
          title: "Subscribe to Newsletter",
        },
      },
      required: ["email"],
    },
  },
};

describe("DynamicSchemaForm", () => {
  it("dynamically renders form fields from JSON Schema traits", () => {
    const onChange = vi.fn();
    render(
      <DynamicSchemaForm
        schema={mockSchema}
        value={{ email: "test@autorix.io", name: { first: "Ada" } }}
        onChange={onChange}
      />
    );

    // Primary identifier badge
    expect(screen.getByText("Primary Identifier")).toBeDefined();

    // Field labels
    expect(screen.getByLabelText(/e-mail address/i)).toBeDefined();
    expect(screen.getByLabelText(/first name/i)).toBeDefined();
    expect(screen.getByLabelText(/last name/i)).toBeDefined();
    expect(screen.getByLabelText(/subscribe to newsletter/i)).toBeDefined();

    // Value binding
    const emailInput = screen.getByLabelText(/e-mail address/i) as HTMLInputElement;
    expect(emailInput.value).toBe("test@autorix.io");

    // Change interaction
    fireEvent.change(emailInput, { target: { value: "new@autorix.io" } });
    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ email: "new@autorix.io" })
    );
  });
});
