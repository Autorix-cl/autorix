import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TupleGrid, Tuple } from "./tuple-grid";

const mockTuples: Tuple[] = [
  { id: "t1", objectType: "document", objectId: "doc_123", relation: "viewer", subjectType: "user", subjectId: "alice" },
  { id: "t2", objectType: "document", objectId: "doc_123", relation: "editor", subjectType: "user", subjectId: "bob" },
  { id: "t3", objectType: "organization", objectId: "org_1", relation: "member", subjectType: "group", subjectId: "devs", subjectRelation: "member" },
];

describe("TupleGrid", () => {
  it("renders a list of tuples formatted correctly", () => {
    render(<TupleGrid tuples={mockTuples} onDelete={vi.fn()} onAdd={vi.fn()} />);
    
    // Check objects
    expect(screen.getAllByText("document:doc_123")).toHaveLength(2);
    expect(screen.getByText("organization:org_1")).toBeInTheDocument();
    
    // Check relations
    expect(screen.getByText("viewer")).toBeInTheDocument();
    expect(screen.getByText("editor")).toBeInTheDocument();
    
    // Check subjects
    expect(screen.getByText("user:alice")).toBeInTheDocument();
    expect(screen.getByText("group:devs#member")).toBeInTheDocument();
  });

  it("selects a row and triggers delete", async () => {
    const user = userEvent.setup();
    const handleDelete = vi.fn().mockResolvedValue(undefined);
    
    render(<TupleGrid tuples={mockTuples} onDelete={handleDelete} onAdd={vi.fn()} />);
    
    // Click the first checkbox (assuming DataTable renders them)
    // We can just query the first row's checkbox
    const checkboxes = screen.getAllByRole("checkbox");
    // [0] is the select-all, [1] is the first row
    await user.click(checkboxes[1]);
    
    // Delete button should appear in the bulk action bar
    const deleteBtn = await screen.findByRole("button", { name: /Delete/i });
    expect(deleteBtn).toBeInTheDocument();
    
    await user.click(deleteBtn);
    
    expect(handleDelete).toHaveBeenCalledWith(["t1"]);
  });
});
