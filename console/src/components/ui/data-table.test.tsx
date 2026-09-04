import { render, screen, fireEvent } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { DataTable } from "./data-table";
import { ColumnDef } from "@tanstack/react-table";

type User = { id: string; name: string; email: string };

const columns: ColumnDef<User>[] = [
  {
    accessorKey: "name",
    header: "Name",
  },
  {
    accessorKey: "email",
    header: "Email",
  },
];

const data: User[] = [
  { id: "1", name: "Alice", email: "alice@example.com" },
  { id: "2", name: "Bob", email: "bob@example.com" },
];

describe("DataTable", () => {
  it("renders table with data", () => {
    render(<DataTable columns={columns} data={data} />);
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.getByText("bob@example.com")).toBeInTheDocument();
  });

  it("handles empty state", () => {
    render(<DataTable columns={columns} data={[]} />);
    expect(screen.getByText("No results.")).toBeInTheDocument();
  });

  it("filters via searchKey", () => {
    render(<DataTable columns={columns} data={data} searchKey="name" />);
    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.change(searchInput, { target: { value: "Alice" } });
    
    expect(screen.getByText("Alice")).toBeInTheDocument();
    expect(screen.queryByText("Bob")).not.toBeInTheDocument();
  });

  it("renders custom emptyState when provided", () => {
    render(
      <DataTable
        columns={columns}
        data={[]}
        emptyState={<div data-testid="custom-empty">Custom SRE Empty State</div>}
      />
    );
    expect(screen.getByTestId("custom-empty")).toBeInTheDocument();
  });

  it("renders rows per page and pagination controls", () => {
    render(<DataTable columns={columns} data={data} defaultPageSize={20} />);
    expect(screen.getByText(/rows per page/i)).toBeInTheDocument();
    expect(screen.getByText(/page 1 of 1/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /previous/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /next/i })).toBeInTheDocument();
  });
});
