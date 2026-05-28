"use client";

import { useEffect, useState } from "react";
import { getPortfolio, listTransactions, type PaperPortfolioResponse, type PaperTransaction } from "@/lib/api/paper";
import { PaperDashboard } from "./PaperDashboard";
import { PaperEquityChart } from "./PaperEquityChart";
import { PaperHoldingsTable } from "./PaperHoldingsTable";
import { PaperRecentTxList } from "./PaperRecentTxList";
import { TradeDialog } from "./TradeDialog";
import { ResetDialog } from "./ResetDialog";

export function PaperPage() {
  const [portfolio, setPortfolio] = useState<PaperPortfolioResponse | null>(null);
  const [txs, setTxs] = useState<PaperTransaction[]>([]);
  const [tradeOpen, setTradeOpen] = useState(false);
  const [resetOpen, setResetOpen] = useState(false);

  async function refresh() {
    const [p, t] = await Promise.all([getPortfolio("90d"), listTransactions(5)]);
    setPortfolio(p);
    setTxs(t.transactions);
  }

  useEffect(() => {
    refresh().catch(() => {
      setPortfolio(null);
      setTxs([]);
    });
  }, []);

  if (!portfolio) {
    return <div className="p-6 text-fg-muted font-mono">로딩…</div>;
  }

  return (
    <div className="p-6 md:p-8 max-w-5xl mx-auto space-y-5">
      <header className="flex items-baseline justify-between">
        <div>
          <h1 className="font-mono text-2xl">📈 Paper Portfolio</h1>
          <p className="text-fg-muted text-sm mt-1">가상 자금으로 매매 시뮬레이션. 수수료·슬리피지 없음.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setTradeOpen(true)}
            className="font-mono text-xs px-3 py-1.5 border border-bb-accent text-bb-accent hover:bg-bb-accent/10"
          >
            + 매매
          </button>
          <button
            onClick={() => setResetOpen(true)}
            className="font-mono text-xs px-3 py-1.5 border border-line text-fg-muted hover:border-fg-muted hover:text-fg"
          >
            ⚙ 리셋
          </button>
        </div>
      </header>

      <PaperDashboard
        account={portfolio.account}
        equity={portfolio.summary.total_equity_krw}
        pnlKrw={portfolio.summary.total_pnl_krw}
        pnlPct={portfolio.summary.total_pnl_pct}
      />

      <div className="border border-line bg-bg-subtle p-5">
        <PaperEquityChart series={portfolio.equity_series} />
      </div>

      <section>
        <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">보유 자산 ({portfolio.holdings.length})</h2>
        <div className="border border-line bg-bg-subtle">
          <PaperHoldingsTable holdings={portfolio.holdings} />
        </div>
      </section>

      <section>
        <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">최근 매매</h2>
        <div className="border border-line bg-bg-subtle p-3">
          <PaperRecentTxList transactions={txs} />
        </div>
      </section>

      <TradeDialog open={tradeOpen} onOpenChange={setTradeOpen} onTraded={refresh} cashBalance={portfolio.account.cash_balance} holdings={portfolio.holdings} />
      <ResetDialog open={resetOpen} onOpenChange={setResetOpen} onReset={refresh} currentInitial={portfolio.account.initial_cash} />
    </div>
  );
}
