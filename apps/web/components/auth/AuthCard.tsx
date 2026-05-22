export function AuthCard({ title, subtitle, children }: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <main className="min-h-screen flex items-center justify-center px-6">
      <div className="w-full max-w-sm border border-line bg-bg-card p-8 space-y-6">
        <header>
          <h1 className="font-mono text-xl">{title}</h1>
          {subtitle && <p className="text-fg-muted text-sm mt-1">{subtitle}</p>}
        </header>
        {children}
      </div>
    </main>
  );
}
