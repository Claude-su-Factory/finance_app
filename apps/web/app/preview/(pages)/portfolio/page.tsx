import { HoldingsTable } from "@/components/portfolio/HoldingsTable";

export default function PreviewPortfolio() {
  return (
    <div className="p-6 md:p-8">
      <header className="flex items-baseline justify-between mb-6">
        <div>
          <h1 className="font-mono text-2xl">포트폴리오</h1>
          <p className="text-fg-muted text-sm mt-1">보유 자산. 시세 지연 15분.</p>
        </div>
      </header>
      <HoldingsTable />
    </div>
  );
}
