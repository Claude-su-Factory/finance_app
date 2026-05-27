import { Sidebar } from "./Sidebar";
import { TopTicker } from "./TopTicker";
import { StatusBar } from "./StatusBar";
import { CommandLauncher } from "@/components/command/CommandLauncher";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col">
      <TopTicker />
      <div className="flex flex-1">
        <Sidebar />
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
      <StatusBar />
      <CommandLauncher />
    </div>
  );
}
