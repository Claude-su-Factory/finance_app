import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AlphaCard } from "./AlphaCard";
import * as api from "@/lib/api/portfolio";

vi.mock("@/lib/api/portfolio", async () => {
  const actual = await vi.importActual<typeof api>("@/lib/api/portfolio");
  return { ...actual, getAlpha: vi.fn() };
});

const mockedGet = vi.mocked(api.getAlpha);

describe("AlphaCard", () => {
  beforeEach(() => mockedGet.mockReset());

  it("renders 3 benchmark rows on success", async () => {
    mockedGet.mockResolvedValueOnce({
      period: "90d", days_requested: 90, days_used: 90, since: "2026-02-27",
      fx_mode: "spot", model: "current_holdings_backward_simulation",
      portfolio: { total_return_pct: 18.94, series: [
        { date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 18.94 },
      ] },
      benchmarks: [
        { key: "kospi", label: "KOSPI", total_return_pct: 3.2, alpha_pp: 15.74,
          series: [{ date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 3.2 }] },
        { key: "sp500", label: "S&P 500", total_return_pct: 6.36, alpha_pp: 12.58,
          series: [{ date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 6.36 }] },
        { key: "kr_us_6040", label: "한미 60/40", total_return_pct: 4.46, alpha_pp: 14.48,
          series: null },
      ],
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/vs KOSPI/)).toBeInTheDocument());
    expect(screen.getByText(/\+15.74%p/)).toBeInTheDocument();
    expect(screen.getByText(/vs S&P 500/)).toBeInTheDocument();
    expect(screen.getByText(/vs 한미 60\/40/)).toBeInTheDocument();
  });

  it("shows account_too_young empty state", async () => {
    mockedGet.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", reason: "account_too_young",
        message: "7일 이상 보유 후 표시됩니다", min_days: 7, current_days: 3 },
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/7일 이상 보유/)).toBeInTheDocument());
    expect(screen.getByText(/가입 3일째/)).toBeInTheDocument();
  });

  it("shows no_holdings empty state with portfolio link", async () => {
    mockedGet.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", reason: "no_holdings",
        message: "보유 자산 추가 후 표시됩니다", min_days: 0, current_days: 0 },
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/보유 자산 추가/)).toBeInTheDocument());
    expect(screen.getByRole("link", { name: /포트폴리오로 이동/ })).toHaveAttribute("href", "/app/portfolio");
  });

  it("re-fetches when period chip clicked", async () => {
    mockedGet.mockResolvedValue({
      period: "90d", days_requested: 90, days_used: 90, since: "2026-02-27",
      fx_mode: "spot", model: "current_holdings_backward_simulation",
      portfolio: { total_return_pct: 0, series: [] },
      benchmarks: [
        { key: "kospi", label: "KOSPI", total_return_pct: 0, alpha_pp: 0, series: [] },
        { key: "sp500", label: "S&P 500", total_return_pct: 0, alpha_pp: 0, series: [] },
        { key: "kr_us_6040", label: "한미 60/40", total_return_pct: 0, alpha_pp: 0, series: null },
      ],
    });
    render(<AlphaCard />);
    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith("90d"));
    fireEvent.click(screen.getByText("1Y"));
    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith("1y"));
  });
});
