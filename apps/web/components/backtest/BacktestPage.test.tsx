import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { BacktestPage } from "./BacktestPage";
import * as btApi from "@/lib/api/backtest";
import * as instApi from "@/lib/api/instruments";

vi.mock("@/lib/api/backtest", async () => {
  const actual = await vi.importActual<typeof btApi>("@/lib/api/backtest");
  return { ...actual, runBacktest: vi.fn() };
});
vi.mock("@/lib/api/instruments", () => ({
  searchInstruments: vi.fn(),
  selectInstrument: vi.fn(),
}));

const mockedRun = vi.mocked(btApi.runBacktest);
const mockedSearch = vi.mocked(instApi.searchInstruments);

const FAKE_INST: instApi.InstrumentResult = {
  id: "11111111-1111-1111-1111-111111111111",
  symbol: "005930",
  exchange: "KRX",
  name: "삼성전자",
  currency: "KRW",
  asset_class: "KR_STOCK",
};

function fakeResult(): btApi.BacktestResult {
  const pts = (a: number, b: number) => [
    { date: "2023-05-29", value: a },
    { date: "2026-05-29", value: b },
  ];
  return {
    clamped_start: "2023-05-29",
    end: "2026-05-29",
    normalized_basket: [
      { instrument_id: FAKE_INST.id, symbol: "005930", name: "삼성전자", weight: 1 },
    ],
    equity_series: pts(10_000_000, 12_000_000),
    contributed_series: pts(10_000_000, 10_000_000),
    benchmarks: {
      kospi: { equity_series: pts(10_000_000, 10_500_000), metrics: { total_return_pct: 5, cagr_pct: 1.64, mdd_pct: -8, volatility_pct: 12, twr_pct: 5 } },
      spx: { equity_series: pts(10_000_000, 11_000_000), metrics: { total_return_pct: 10, cagr_pct: 3.2, mdd_pct: -6, volatility_pct: 14, twr_pct: 10 } },
      sixty_forty: { equity_series: pts(10_000_000, 10_700_000), metrics: { total_return_pct: 7, cagr_pct: 2.3, mdd_pct: -5, volatility_pct: 9, twr_pct: 7 } },
    },
    metrics: {
      total_return_pct: 20,
      cagr_pct: 6.27,
      mdd_pct: -10,
      volatility_pct: 15,
      excess_vs_6040_pct: 13,
      total_contributed: 10_000_000,
      final_equity: 12_000_000,
    },
    coverage_warnings: [],
  };
}

async function addLegAndRun() {
  fireEvent.change(screen.getByPlaceholderText("＋ 종목 추가 (검색)"), {
    target: { value: "삼성" },
  });
  const pick = await screen.findByText("삼성전자", {}, { timeout: 2000 });
  fireEvent.click(pick.closest("button")!);
  fireEvent.change(await screen.findByLabelText("005930 비중"), {
    target: { value: "100" },
  });
  fireEvent.click(screen.getByText("백테스트 실행"));
}

describe("BacktestPage", () => {
  beforeEach(() => {
    mockedRun.mockReset();
    mockedSearch.mockReset();
    mockedSearch.mockResolvedValue([FAKE_INST]);
  });

  it("shows empty state before running", () => {
    render(<BacktestPage />);
    expect(
      screen.getByText(/바스켓과 전략을 설정하고 실행하세요/),
    ).toBeInTheDocument();
  });

  it("renders metrics and compare table on success", async () => {
    mockedRun.mockResolvedValueOnce(fakeResult());
    render(<BacktestPage />);
    await addLegAndRun();
    await waitFor(() =>
      expect(screen.getByText("초과수익 vs 60/40")).toBeInTheDocument(),
    );
    expect(screen.getByText("+20.00%")).toBeInTheDocument(); // 전략 총수익률 카드(부호 표기)
    expect(screen.getByText("내 전략")).toBeInTheDocument();
    expect(screen.getByText("KOSPI")).toBeInTheDocument();
    expect(mockedRun).toHaveBeenCalledWith(
      expect.objectContaining({
        period: "3Y",
        initial_cash: 10_000_000,
        basket: [{ instrument_id: FAKE_INST.id, weight: 100 }],
      }),
    );
  });

  it("shows error message when API returns 422", async () => {
    mockedRun.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", message: "데이터가 부족합니다", min_days: 30, current_days: 12 },
    });
    render(<BacktestPage />);
    await addLegAndRun();
    await waitFor(() =>
      expect(screen.getByText("데이터가 부족합니다")).toBeInTheDocument(),
    );
  });
});
