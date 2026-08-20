import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@/lib/i18n";
import { LoadingState } from "./loading-state";
import { EmptyState } from "./empty-state";
import { ErrorState } from "./error-state";
import { NotConnectedState } from "./not-connected-state";
import type { ApiError } from "@/lib/api/client";

function withI18n(ui: React.ReactElement) {
  return render(<I18nProvider>{ui}</I18nProvider>);
}

describe("LoadingState", () => {
  it("renders a status role so assistive tech announces it", () => {
    withI18n(<LoadingState />);
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders a custom label when given one", () => {
    withI18n(<LoadingState label="Fetching identities..." />);
    expect(screen.getByText("Fetching identities...")).toBeInTheDocument();
  });
});

describe("EmptyState", () => {
  it("renders default title and description", () => {
    withI18n(<EmptyState />);
    expect(screen.getByText(/nothing here yet/i)).toBeInTheDocument();
  });

  it("renders custom title/description and an action when given", () => {
    withI18n(
      <EmptyState title="No identities" description="Create your first one." action={<button>Create</button>} />,
    );
    expect(screen.getByText("No identities")).toBeInTheDocument();
    expect(screen.getByText("Create your first one.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Create" })).toBeInTheDocument();
  });
});

describe("ErrorState", () => {
  it("renders an alert role with the error message", () => {
    const error: ApiError = { kind: "engine-error", message: "internal server error" };
    withI18n(<ErrorState error={error} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText("internal server error")).toBeInTheDocument();
  });

  it("renders a retry affordance and calls onRetry when clicked", async () => {
    const onRetry = vi.fn();
    const error: ApiError = { kind: "engine-error", message: "boom" };
    withI18n(<ErrorState error={error} onRetry={onRetry} />);

    const button = screen.getByRole("button", { name: /retry/i });
    await userEvent.click(button);

    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("does not render a retry button when onRetry is not given", () => {
    const error: ApiError = { kind: "engine-error", message: "boom" };
    withI18n(<ErrorState error={error} />);
    expect(screen.queryByRole("button", { name: /retry/i })).not.toBeInTheDocument();
  });

  it("shows an unauthorized-specific message for a 401/403 error", () => {
    const error: ApiError = { kind: "unauthorized", message: "no session", status: 401 };
    withI18n(<ErrorState error={error} />);
    expect(screen.getByText(/session doesn't have access/i)).toBeInTheDocument();
  });
});

describe("NotConnectedState", () => {
  it("names the engine in both the title and description", () => {
    withI18n(<NotConnectedState engineName="Vulcan" />);
    expect(screen.getByText(/vulcan is not reachable/i)).toBeInTheDocument();
    expect(screen.getByText(/could not reach vulcan/i)).toBeInTheDocument();
  });

  it("renders a retry affordance and calls onRetry when clicked", async () => {
    const onRetry = vi.fn();
    withI18n(<NotConnectedState engineName="Vulcan" onRetry={onRetry} />);

    await userEvent.click(screen.getByRole("button", { name: /retry/i }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
