import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { MfaPanel } from "./mfa-panel";

describe("MfaPanel", () => {
  const enrolledFactors = ["Authenticator App", "WebAuthn (TouchID)"];

  it("renders the list of enrolled factors", () => {
    render(<MfaPanel factors={enrolledFactors} onGenerateRecovery={vi.fn()} />);
    
    expect(screen.getByText("Authenticator App")).toBeInTheDocument();
    expect(screen.getByText("WebAuthn (TouchID)")).toBeInTheDocument();
  });

  it("calls onGenerateRecovery and displays warning when button is clicked", async () => {
    const user = userEvent.setup();
    const handleGenerate = vi.fn().mockResolvedValue(["abc-123", "def-456"]);
    
    render(<MfaPanel factors={enrolledFactors} onGenerateRecovery={handleGenerate} />);
    
    const generateBtn = screen.getByRole("button", { name: /Generate Recovery Codes/i });
    await user.click(generateBtn);
    
    expect(handleGenerate).toHaveBeenCalledTimes(1);
    
    // Check that the UI reacted (warning message appears when codes are generated)
    await waitFor(() => {
      expect(screen.getByText(/Warning: These codes will only be shown once/i)).toBeInTheDocument();
    });
  });
});
