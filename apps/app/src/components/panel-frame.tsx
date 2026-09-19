// Frames an authenticated panel with the brand bar, optional navigation, logout, and the greeting heading.

import type { ReactNode } from "react";

import { authCopy } from "../auth/copy";

type PanelFrameProps = { eyebrow: string; name: string; lede: string; nav?: ReactNode; onLogout: () => void; children: ReactNode };

export function PanelFrame({ eyebrow, name, lede, nav, onLogout, children }: PanelFrameProps) {
  return <>
    <header className="panel-header">
      <div className="panel-bar">
        <p className="brand">{authCopy.brand}</p>
        {nav}
        <button className="btn btn-ghost btn-sm" onClick={onLogout}>{authCopy.logout}</button>
      </div>
    </header>
    <main className="panel-shell">
      <p className="eyebrow">{eyebrow}</p>
      <h1 className="panel-title">Cześć, <span className="marker">{name}</span>.</h1>
      <p className="panel-lede">{lede}</p>
      {children}
    </main>
  </>;
}
