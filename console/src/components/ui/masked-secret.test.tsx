import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { MaskedSecret, CopyableIdentifier } from "./masked-secret";

describe("MaskedSecret", () => {
  it("renders masked text by default and reveals on click", () => {
    const onReveal = vi.fn();
    render(<MaskedSecret secret="super_secret_macaroon_123" label="Macaroon" onReveal={onReveal} />);

    // Default masked
    expect(screen.getByText("•".repeat(24))).toBeDefined();
    expect(screen.queryByText("super_secret_macaroon_123")).toBeNull();

    // Click reveal
    const revealBtn = screen.getByRole("button", { name: /reveal secret/i });
    fireEvent.click(revealBtn);

    expect(onReveal).toHaveBeenCalledTimes(1);
    expect(screen.getByText("super_secret_macaroon_123")).toBeDefined();

    // Click mask again
    const maskBtn = screen.getByRole("button", { name: /mask secret/i });
    fireEvent.click(maskBtn);
    expect(screen.getByText("•".repeat(24))).toBeDefined();
  });

  it("auto-masks after timeout", () => {
    vi.useFakeTimers();
    render(<MaskedSecret secret="temp_secret" autoMaskDelayMs={1000} />);

    fireEvent.click(screen.getByRole("button", { name: /reveal secret/i }));
    expect(screen.getByText("temp_secret")).toBeDefined();

    act(() => {
      vi.advanceTimersByTime(1100);
    });

    expect(screen.getByText("•".repeat(24))).toBeDefined();
    vi.useRealTimers();
  });
});

describe("CopyableIdentifier", () => {
  it("renders truncated value and copies to clipboard", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.assign(navigator, {
      clipboard: { writeText },
    });

    render(<CopyableIdentifier value="0/16B23F899A1234567890" label="WAL LSN" truncateLength={10} />);
    const btn = screen.getByRole("button", { name: /copy wal lsn/i });

    expect(btn.textContent).toContain("...");

    await act(async () => {
      fireEvent.click(btn);
    });

    expect(writeText).toHaveBeenCalledWith("0/16B23F899A1234567890");
  });
});
