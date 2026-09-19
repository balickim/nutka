// Reports the result of a write. Holds no server state, so it stays outside the query cache.

import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";

type Tone = "success" | "danger";
type Notice = { id: number; tone: Tone; message: string; undo?: () => void };
type ToastApi = { notify: (message: string, tone?: Tone, undo?: () => void) => void };

const ToastContext = createContext<ToastApi | null>(null);
const DISMISS_AFTER_MS = 6000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [notices, setNotices] = useState<Notice[]>([]);
  const dismiss = useCallback((id: number) => setNotices((current) => current.filter((notice) => notice.id !== id)), []);
  const notify = useCallback((message: string, tone: Tone = "success", undo?: () => void) => {
    const id = Date.now() + Math.random();
    setNotices((current) => [...current, { id, tone, message, undo }]);
    window.setTimeout(() => dismiss(id), DISMISS_AFTER_MS);
  }, [dismiss]);
  const api = useMemo(() => ({ notify }), [notify]);
  return <ToastContext.Provider value={api}>{children}<ToastTray notices={notices} onDismiss={dismiss} /></ToastContext.Provider>;
}

export function useToast(): ToastApi {
  const api = useContext(ToastContext);
  if (!api) throw new Error("useToast requires ToastProvider");
  return api;
}

function ToastTray({ notices, onDismiss }: { notices: Notice[]; onDismiss: (id: number) => void }) {
  if (notices.length === 0) return null;
  return <div className="toast-tray" role="status" aria-live="polite">{notices.map((notice) => <div className={`toast toast-${notice.tone}`} key={notice.id}>
    <span>{notice.message}</span>
    {notice.undo ? <button className="text-button" onClick={() => { notice.undo?.(); onDismiss(notice.id); }}>Cofnij</button> : null}
    <button className="text-button" aria-label="Zamknij powiadomienie" onClick={() => onDismiss(notice.id)}>×</button>
  </div>)}</div>;
}
