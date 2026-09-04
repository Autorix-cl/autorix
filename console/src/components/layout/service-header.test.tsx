import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { ServiceHeader } from "./service-header";
import { Users, Shield } from "lucide-react";
import { EnvironmentProvider } from "@/lib/environment/environment-context";

const wrapper = ({ children }: { children: React.ReactNode }) => (
  <EnvironmentProvider>{children}</EnvironmentProvider>
);

describe("ServiceHeader", () => {
  it("renders cloud breadcrumb, title, badge, and metrics strip", () => {
    render(
      <ServiceHeader
        serviceName="Ego Identity"
        title="Identities & Credentials"
        description="Manage user traits and passkeys"
        icon={Users}
        statusText="REST Headless Active"
        metrics={[
          { label: "Total Identities", value: "142", hint: "Active subjects" },
          { label: "Engine Mesh", value: "mTLS", hint: "Port 4433", icon: Shield },
        ]}
      />,
      { wrapper }
    );

    // Breadcrumb
    expect(screen.getByText("Autorix Cloud")).toBeInTheDocument();
    expect(screen.getByText("Ego Identity")).toBeInTheDocument();

    // Title & Status
    expect(screen.getByText("Identities & Credentials")).toBeInTheDocument();
    expect(screen.getByText("REST Headless Active")).toBeInTheDocument();

    // Metrics
    expect(screen.getByText(/total identities/i)).toBeInTheDocument();
    expect(screen.getByText("142")).toBeInTheDocument();
    expect(screen.getByText(/engine mesh/i)).toBeInTheDocument();
    expect(screen.getByText("mTLS")).toBeInTheDocument();
  });
});
