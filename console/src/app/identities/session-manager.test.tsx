import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { SessionManager } from "./session-manager";

const mockSessions = [
  {
    id: "sess_1",
    ip: "192.168.1.10",
    userAgent: "Chrome on macOS",
    lastAccess: "2023-10-10 10:00:00",
  },
  {
    id: "sess_2",
    ip: "10.0.0.5",
    userAgent: "Safari on iOS",
    lastAccess: "2023-10-11 15:30:00",
  }
];

describe("SessionManager", () => {
  it("renders a list of sessions with their details", () => {
    render(<SessionManager sessions={mockSessions} onRevoke={vi.fn()} />);
    
    expect(screen.getByText("192.168.1.10")).toBeInTheDocument();
    expect(screen.getByText("Chrome on macOS")).toBeInTheDocument();
  });

  it("calls onRevoke with the correct session id when clicked", async () => {
    const user = userEvent.setup();
    const handleRevoke = vi.fn();
    render(<SessionManager sessions={mockSessions} onRevoke={handleRevoke} />);
    
    // Find the revoke button for sess_1
    const revokeBtn = screen.getByRole("button", { name: "Revoke session for 192.168.1.10" });
    await user.click(revokeBtn);
    
    expect(handleRevoke).toHaveBeenCalledWith("sess_1");
  });
});
