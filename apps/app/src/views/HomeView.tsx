import { useEffect, useState } from "react";

import { checkHealth, type HealthState } from "../api/health";
import { authCopy } from "../auth/copy";
import { bootstrapAuth, getAuthState, getLearnerDisplayName, logout, subscribe } from "../auth/auth";
import { router } from "../router";

export function HomeView() {
  const authState = useAuthState();
  const [health, setHealth] = useState<HealthState>("checking");
  const [loggingOut, setLoggingOut] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    void checkHealth(controller.signal).then(setHealth).catch(() => undefined);
    return () => controller.abort();
  }, []);
  useEffect(() => {
    if (authState.status === "ready" && !authState.record) {
      void router.navigate({ to: "/login" });
    }
  }, [authState.status, authState.record]);
  if (authState.status === "error") return <main className="center-shell"><section className="status-card" role="alert"><p className="eyebrow">nutka / sesja</p><h1>{authCopy.unavailable}</h1><button className="secondary-button" onClick={() => void bootstrapAuth()}>{authCopy.retry}</button></section></main>;
  if (!authState.record) return null;
  async function handleLogout() {
    setLoggingOut(true);
    await logout();
    await router.navigate({ to: "/login" });
  }
  return (
    <main className="home-shell">
      <header className="home-header"><p className="wordmark">{authCopy.brand}</p><button className="text-button" onClick={() => void handleLogout()} disabled={loggingOut}>{authCopy.logout}</button></header>
      <section className="home-card">
        <p className="eyebrow">{authCopy.learnerHome}</p>
        <h1>Cześć, {getLearnerDisplayName(authState.record)}.</h1>
        <p className="home-description">{authCopy.learnerHomeDescription}</p>
        <div className="learner-details"><span>{authCopy.signedInAs}</span><strong>{authState.record.email}</strong></div>
        <div className={`health-pill health-${health}`}><span className="health-dot" />{health === "checking" ? authCopy.backendChecking : health === "ready" ? authCopy.backendReady : authCopy.backendUnavailable}</div>
      </section>
    </main>
  );
}

function useAuthState() {
  const [, rerender] = useState(0);
  useEffect(() => subscribe(() => rerender((value) => value + 1)), []);
  return getAuthState();
}
