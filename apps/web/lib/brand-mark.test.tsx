import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { SquareMark } from "./brand-mark";

describe("SquareMark", () => {
  it("renders the Q glyph", () => {
    render(<SquareMark size={32} />);
    expect(screen.getByText("Q")).toBeInTheDocument();
  });

  it("scales font-size with the size prop", () => {
    const { container } = render(<SquareMark size={512} />);
    const root = container.firstElementChild as HTMLElement;
    expect(root.style.width).toBe("512px");
    expect(root.style.height).toBe("512px");
    expect(root.style.display).toBe("flex");
  });
});
