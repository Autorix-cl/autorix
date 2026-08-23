import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { CheckSimulator } from "./check-simulator";

describe("CheckSimulator", () => {
  it("renders the simulation form and default idle state", () => {
    render(<CheckSimulator onCheck={vi.fn()} />);
    
    expect(screen.getByLabelText(/Subject/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Relation/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Object/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Check Access/i })).toBeInTheDocument();
  });

  it("displays ALLOWED and trace when check succeeds", async () => {
    const user = userEvent.setup();
    const handleCheck = vi.fn().mockResolvedValue({
      allowed: true,
      trace: ["user:alice is member of group:admin", "group:admin is editor of document:1"]
    });
    
    render(<CheckSimulator onCheck={handleCheck} />);
    
    await user.type(screen.getByLabelText(/Subject/i), "user:alice");
    await user.type(screen.getByLabelText(/Relation/i), "editor");
    await user.type(screen.getByLabelText(/Object/i), "document:1");
    
    await user.click(screen.getByRole("button", { name: /Check Access/i }));
    
    expect(handleCheck).toHaveBeenCalledWith({
      subject: "user:alice",
      relation: "editor",
      object: "document:1"
    });
    
    await waitFor(() => {
      expect(screen.getByText("ALLOWED")).toBeInTheDocument();
      expect(screen.getByText("user:alice is member of group:admin")).toBeInTheDocument();
    });
  });
});
