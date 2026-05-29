import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import LandingPage from "./page";

describe("LandingPage", () => {
  it("renders headline + CTAs", () => {
    render(<LandingPage />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("한국·미국 자산을 한 화면에");
    expect(screen.getAllByText(/무료로 시작/).length).toBeGreaterThan(0);
    expect(screen.getByText("로그인")).toBeInTheDocument();
  });
});
