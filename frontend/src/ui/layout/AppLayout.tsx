import type { PropsWithChildren } from "react";

interface AppLayoutProps extends PropsWithChildren {
  tone?: "default" | "loading";
}

export function AppLayout({ children, tone = "default" }: AppLayoutProps) {
  return (
    <main className={`app-shell app-shell--${tone}`}>
      <div className="app-shell__backdrop" />
      <div className="app-shell__content">{children}</div>
    </main>
  );
}
