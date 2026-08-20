import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("sonner", () => ({
  toast: {
    success: vi.fn(),
    error: vi.fn(),
  },
}));

import { toast } from "sonner";
import { toastSuccess, toastApiError } from "./toast";
import type { ApiError } from "./api/client";

beforeEach(() => {
  vi.clearAllMocks();
});

describe("toastSuccess", () => {
  it("shows a success toast naming what happened", () => {
    toastSuccess("Identity created");
    expect(toast.success).toHaveBeenCalledWith("Identity created");
  });
});

describe("toastApiError", () => {
  it("shows the error message plus a fix for engine-unreachable", () => {
    const error: ApiError = { kind: "engine-unreachable", message: "network error" };
    toastApiError(error);
    expect(toast.error).toHaveBeenCalledWith(
      "network error",
      expect.objectContaining({ description: expect.stringMatching(/running and reachable/i) }),
    );
  });

  it("shows a fix naming re-authentication for unauthorized", () => {
    const error: ApiError = { kind: "unauthorized", message: "no session" };
    toastApiError(error);
    expect(toast.error).toHaveBeenCalledWith(
      "no session",
      expect.objectContaining({ description: expect.stringMatching(/sign in again/i) }),
    );
  });

  it("shows a fix naming the highlighted fields for validation errors", () => {
    const error: ApiError = { kind: "validation", message: "email is required" };
    toastApiError(error);
    expect(toast.error).toHaveBeenCalledWith(
      "email is required",
      expect.objectContaining({ description: expect.stringMatching(/fix the highlighted fields/i) }),
    );
  });

  it("allows a custom fix message to override the default", () => {
    const error: ApiError = { kind: "engine-error", message: "boom" };
    toastApiError(error, "Custom fix instructions");
    expect(toast.error).toHaveBeenCalledWith(
      "boom",
      expect.objectContaining({ description: "Custom fix instructions" }),
    );
  });
});
