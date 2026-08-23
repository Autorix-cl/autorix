import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SchemaEditor } from "./schema-editor";

describe("SchemaEditor", () => {
  const initialSchema = `definition user {}
definition document {
  relation viewer: user
  relation editor: user
  permission view = viewer + editor
}`;

  it("renders in read-only mode by default", () => {
    render(<SchemaEditor schema={initialSchema} onSave={vi.fn()} />);
    
    // Should show the Edit button
    expect(screen.getByRole("button", { name: /Edit Schema/i })).toBeInTheDocument();
    
    // Should not show the Save button
    expect(screen.queryByRole("button", { name: /Save & Validate/i })).not.toBeInTheDocument();
  });

  it("switches to edit mode and calls onSave when submitted", async () => {
    const user = userEvent.setup();
    const handleSave = vi.fn().mockResolvedValue(undefined); // Success
    
    render(<SchemaEditor schema={initialSchema} onSave={handleSave} />);
    
    // Enter edit mode
    await user.click(screen.getByRole("button", { name: /Edit Schema/i }));
    
    // Save button should appear
    const saveBtn = screen.getByRole("button", { name: /Save & Validate/i });
    expect(saveBtn).toBeInTheDocument();
    
    // Click save
    await user.click(saveBtn);
    
    expect(handleSave).toHaveBeenCalledTimes(1);
    // Should expect the mock schema to be passed, but since we can't easily type into CodeMirror in jsdom, we check if it sent the initial value
    expect(handleSave).toHaveBeenCalledWith(initialSchema);
    
    // Should return to read-only mode after save
    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Edit Schema/i })).toBeInTheDocument();
    });
  });
});
