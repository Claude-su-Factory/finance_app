"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import type { InstrumentResult } from "@/lib/api/instruments";

export type DraftHolding = {
  instrument: InstrumentResult;
  quantity: number;
  avg_cost: number;
};

export function HoldingsStep({
  value, onChange, onNext, onSkip,
}: {
  value: DraftHolding[];
  onChange: (v: DraftHolding[]) => void;
  onNext: () => void;
  onSkip: () => void;
}) {
  const [inst, setInst] = useState<InstrumentResult | null>(null);
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [err, setErr] = useState<string | null>(null);

  function addOne() {
    if (!inst) { setErr("종목을 선택해주세요"); return; }
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    if (value.find((d) => d.instrument.id === inst.id)) { setErr("이미 추가된 종목입니다"); return; }
    onChange([...value, { instrument: inst, quantity: q, avg_cost: c }]);
    setInst(null); setQuantity(""); setAvgCost(""); setErr(null);
  }

  return (
    <div className="space-y-4">
      <h2 className="font-mono text-lg">첫 보유 자산을 1~3개 추가하세요</h2>
      <p className="text-fg-muted text-xs font-mono">건너뛰면 빈 포트폴리오로 시작합니다. 나중에 언제든 추가할 수 있습니다.</p>

      <div className="border-l-2 border-bb-warn/60 bg-bb-warn/5 px-3 py-2 text-xs text-fg-muted leading-relaxed">
        <span className="font-mono text-bb-warn">i</span>{" "}
        본 서비스는 증권사 자동 연동 없이 본인이 직접 기록하는 <span className="text-fg">개인 분석 도구</span>입니다.
        실제 보유 여부는 검증하지 않으며, 분석 정확성은 입력 정확성에 따릅니다.
      </div>

      {value.length > 0 && (
        <ul className="space-y-1 font-mono text-sm border border-line p-2">
          {value.map((d, i) => (
            <li key={i} className="flex gap-2">
              <span className="flex-1">{d.instrument.symbol} — {d.instrument.name}</span>
              <span className="tabular-nums text-xs text-fg-muted">{d.quantity} @ {d.avg_cost.toLocaleString()} {d.instrument.currency}</span>
              <button type="button" onClick={() => onChange(value.filter((_, j) => j !== i))} className="text-xs text-bb-down">×</button>
            </li>
          ))}
        </ul>
      )}

      {value.length < 3 && (
        <div className="space-y-3 border border-line p-3">
          <div>
            <Label className="text-xs font-mono">종목</Label>
            <InstrumentSearchInput onSelect={setInst} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs font-mono">수량</Label>
              <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} type="number" step="any" />
            </div>
            <div>
              <Label className="text-xs font-mono">평단가 ({inst?.currency ?? "통화"})</Label>
              <Input value={avgCost} onChange={(e) => setAvgCost(e.target.value)} type="number" step="any" />
            </div>
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
          <Button variant="ghost" onClick={addOne}>+ 추가 ({value.length}/3)</Button>
        </div>
      )}

      <div className="flex gap-2 justify-end">
        <Button variant="ghost" onClick={onSkip}>건너뛰기</Button>
        <Button onClick={onNext}>다음 →</Button>
      </div>
    </div>
  );
}
