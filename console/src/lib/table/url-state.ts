export interface TableQueryState {
  page: number;
  pageSize: number;
  sortBy?: string;
  sortOrder: "asc" | "desc";
  search: string;
  filters: Record<string, string>;
}

export function serializeTableParams(state: Partial<TableQueryState>): URLSearchParams {
  const params = new URLSearchParams();

  if (state.page && state.page > 1) {
    params.set("page", state.page.toString());
  }
  if (state.pageSize && state.pageSize !== 10) {
    params.set("pageSize", state.pageSize.toString());
  }
  if (state.sortBy) {
    params.set("sortBy", state.sortBy);
  }
  if (state.sortOrder) {
    params.set("sortOrder", state.sortOrder);
  }
  if (state.search && state.search.trim()) {
    params.set("q", state.search.trim());
  }
  if (state.filters) {
    for (const [key, value] of Object.entries(state.filters)) {
      if (value) {
        params.set(`f_${key}`, value);
      }
    }
  }

  return params;
}

export function deserializeTableParams(searchParams: URLSearchParams): TableQueryState {
  const page = parseInt(searchParams.get("page") || "1", 10) || 1;
  const pageSize = parseInt(searchParams.get("pageSize") || "10", 10) || 10;
  const sortBy = searchParams.get("sortBy") || undefined;
  const sortOrder = searchParams.get("sortOrder") === "desc" ? "desc" : "asc";
  const search = searchParams.get("q") || "";

  const filters: Record<string, string> = {};
  searchParams.forEach((value, key) => {
    if (key.startsWith("f_")) {
      const filterKey = key.slice(2);
      filters[filterKey] = value;
    }
  });

  return {
    page,
    pageSize,
    sortBy,
    sortOrder,
    search,
    filters,
  };
}
