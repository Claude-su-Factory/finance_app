"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { TotalAssetCard } from "./TotalAssetCard";
import { AllocationDonut } from "./AllocationDonut";
import { Skeleton } from "@/components/ui/skeleton";

export function HomeDashboard() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);

  useEffect(() => {
    listHoldings().then(setHoldings).catch(() => setHoldings([]));
  }, []);

  if (holdings === null) {
    return (
      <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32" />)}
      </div>
    );
  }

  return (
    <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <TotalAssetCard holdings={holdings} />
      <AllocationDonut holdings={holdings} />
      {/* W3-T14에서 상위5·마켓·관심종목·브리핑 placeholder 4 카드 추가 */}
    </div>
  );
}
