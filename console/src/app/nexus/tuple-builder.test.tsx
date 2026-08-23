import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { TupleBuilder } from "./tuple-builder";

describe("TupleBuilder", () => {
  it("renders all required input fields", () => {
    render(<TupleBuilder onSubmit={vi.fn()} onCancel={vi.fn()} />);
    
    expect(screen.getByLabelText(/Object Type/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Object ID/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Relation \*/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Subject Type/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Subject ID/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Subject Relation/i)).toBeInTheDocument();
  });

  it("submits the correct tuple data", async () => {
    const user = userEvent.setup();
    const handleSubmit = vi.fn().mockResolvedValue(undefined);
    
    render(<TupleBuilder onSubmit={handleSubmit} onCancel={vi.fn()} />);
    
    await user.type(screen.getByLabelText(/Object Type/i), "document");
    await user.type(screen.getByLabelText(/Object ID/i), "doc_1");
    await user.type(screen.getByLabelText(/^Relation \*/i), "viewer");
    await user.type(screen.getByLabelText(/Subject Type/i), "group");
    await user.type(screen.getByLabelText(/Subject ID/i), "engineering");
    await user.type(screen.getByLabelText(/Subject Relation/i), "member");
    
    await user.click(screen.getByRole("button", { name: /Create Tuple/i }));
    
    expect(handleSubmit).toHaveBeenCalledWith({
      objectType: "document",
      objectId: "doc_1",
      relation: "viewer",
      subjectType: "group",
      subjectId: "engineering",
      subjectRelation: "member"
    });
  });
});
