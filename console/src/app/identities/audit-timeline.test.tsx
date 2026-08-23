import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { AuditTimeline } from "./audit-timeline";

const mockEvents = [
  {
    id: "evt_1",
    action: "user.suspended",
    actor: "admin@autorix.com",
    timestamp: "2023-10-12 09:00:00",
  },
  {
    id: "evt_2",
    action: "mfa.removed",
    actor: "system",
    timestamp: "2023-10-11 14:20:00",
  }
];

describe("AuditTimeline", () => {
  it("renders a list of audit events correctly", () => {
    render(<AuditTimeline events={mockEvents} />);
    
    // Should render actions
    expect(screen.getByText("user.suspended")).toBeInTheDocument();
    expect(screen.getByText("mfa.removed")).toBeInTheDocument();
    
    // Should render actors
    expect(screen.getByText(/admin@autorix\.com/)).toBeInTheDocument();
    expect(screen.getByText(/system/)).toBeInTheDocument();
  });

  it("renders an empty state when there are no events", () => {
    render(<AuditTimeline events={[]} />);
    expect(screen.getByText("No recent activity.")).toBeInTheDocument();
  });
});
