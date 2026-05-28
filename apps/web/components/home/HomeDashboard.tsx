"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { TotalAssetCard } from "./TotalAssetCard";
import { AllocationDonut } from "./AllocationDonut";
import { TopHoldingsCard } from "./TopHoldingsCard";
import { MarketWidgetsCard } from "./MarketWidgetsCard";
import { WatchlistMiniCard } from "./WatchlistMiniCard";
import { BriefingCard } from "./BriefingCard";
import { AlphaCard } from "./AlphaCard";
import { Skeleton } from "@/components/ui/skeleton";

export function HomeDashboard() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);

  useEffect(() => {
    listHoldings().then(setHoldings).catch(() => setHoldings([]));
  }, []);

  if (holdings === null) {
    return (
      <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 7 }).map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
    );
  }

  return (
    <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {/* 1행: 총자산 · 도넛 · 알파 */}
      <TotalAssetCard holdings={holdings} />
      <AllocationDonut holdings={holdings} />
      <AlphaCard />
      {/* 2행: 상위5 · 마켓 · 관심종목 */}
      <TopHoldingsCard holdings={holdings} />
      <MarketWidgetsCard />
      <WatchlistMiniCard />
      {/* 3행: 브리핑 (가로 와이드) */}
      <div className="lg:col-span-3">
        <BriefingCard />
      </div>
    </div>
  );
}
