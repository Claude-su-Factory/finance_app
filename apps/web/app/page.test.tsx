import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import LandingPage from "./page";

describe("LandingPage", () => {
  it("renders headline + CTAs", () => {
    render(<LandingPage />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Portfolio Intelligence Terminal");
    expect(screen.getByText("가입하기")).toBeInTheDocument();
    expect(screen.getByText("로그인")).toBeInTheDocument();
  });
});
