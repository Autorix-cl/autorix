import { render } from "@testing-library/react";
import { describe, it, expect, vi } from "vitest";
import { CodeEditor } from "./code-editor";

describe("CodeEditor", () => {
  it("renders correctly with JSON language", () => {
    const onChange = vi.fn();
    const { container } = render(
      <CodeEditor value='{"test": true}' onChange={onChange} language="json" />
    );
    // Codemirror wrapper renders correctly
    const wrapper = container.querySelector(".overflow-hidden.rounded-md.border");
    expect(wrapper).toBeInTheDocument();
  });
});
