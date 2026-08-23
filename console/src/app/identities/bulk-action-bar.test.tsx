import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { BulkActionBar } from "./bulk-action-bar";

describe("BulkActionBar", () => {
  it("does not render when selectedCount is 0", () => {
    const { container } = render(<BulkActionBar selectedCount={0} onSuspend={vi.fn()} onClearSelection={vi.fn()} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders with correct count and calls onSuspend", async () => {
    const user = userEvent.setup();
    const handleSuspend = vi.fn();
    
    render(<BulkActionBar selectedCount={3} onSuspend={handleSuspend} onClearSelection={vi.fn()} />);
    
    expect(screen.getByText("3 users selected")).toBeInTheDocument();
    
    const suspendBtn = screen.getByRole("button", { name: /Suspend/i });
    await user.click(suspendBtn);
    
    expect(handleSuspend).toHaveBeenCalledTimes(1);
  });
});
