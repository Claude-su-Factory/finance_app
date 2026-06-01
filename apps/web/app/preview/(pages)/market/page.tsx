import { KRIndicesCard } from "@/components/market/KRIndicesCard";
import { USIndicesCard } from "@/components/market/USIndicesCard";
import { FxCard } from "@/components/market/FxCard";
import { IndicatorsCard } from "@/components/market/IndicatorsCard";
import { WatchlistEditorCard } from "@/components/market/WatchlistEditorCard";

export default function PreviewMarket() {
  return (
    <div className="p-6 md:p-8 space-y-4">
      <header className="flex items-baseline justify-between mb-2">
        <div>
          <h1 className="font-mono text-2xl">마켓</h1>
          <p className="text-fg-muted text-sm mt-1">지수·환율·지표. 시세 지연 15분.</p>
        </div>
      </header>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <KRIndicesCard />
        <USIndicesCard />
        <FxCard pair={["USD", "KRW"]} title="USD/KRW" />
        <FxCard pair={["EUR", "KRW"]} title="EUR/KRW" />
        <FxCard pair={["JPY", "KRW"]} title="JPY/KRW" />
        <IndicatorsCard code="DFF" title="Fed Funds Rate" unit="%" />
        <IndicatorsCard code="DGS10" title="US 10Y Treasury" unit="%" />
        <IndicatorsCard code="722Y001" title="BOK 기준금리" unit="%" />
        <WatchlistEditorCard />
      </div>
    </div>
  );
}
