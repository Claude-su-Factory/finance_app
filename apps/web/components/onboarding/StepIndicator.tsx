export function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <div className="flex gap-1 px-4 py-2">
      {Array.from({ length: total }).map((_, i) => (
        <div
          key={i}
          className={`h-[2px] flex-1 ${i < current ? "bg-bb-accent" : "bg-line"}`}
        />
      ))}
    </div>
  );
}
