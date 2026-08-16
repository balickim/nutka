import { useEffect, useState } from "react";

type HealthState = "checking" | "ready" | "unavailable";

const apiUrl = import.meta.env.VITE_API_URL ?? "http://127.0.0.1:8090";
const landingUrl = import.meta.env.VITE_LANDING_URL ?? "http://127.0.0.1:4321";

export default function App() {
  const [health, setHealth] = useState<HealthState>("checking");

  useEffect(() => {
    const controller = new AbortController();

    async function checkHealth() {
      try {
        const response = await fetch(`${apiUrl}/api/health`, { signal: controller.signal });
        setHealth(response.ok ? "ready" : "unavailable");
      } catch (error) {
        if (!(error instanceof DOMException && error.name === "AbortError")) {
          setHealth("unavailable");
        }
      }
    }

    void checkHealth();
    return () => controller.abort();
  }, []);

  const statusMessage = {
    checking: "Checking your practice room…",
    ready: "Practice room ready",
    unavailable: "Practice room is offline",
  }[health];

  return (
    <main className="app-shell">
      <header>
        <a className="brand" href={landingUrl}>nutka<span>•</span></a>
        <span className={`status status-${health}`}>{statusMessage}</span>
      </header>
      <section className="welcome-card">
        <p className="eyebrow">YOUR LEARNER SPACE</p>
        <h1>Welcome to your music room.</h1>
        <p>Your lessons, practice prompts, and musical milestones will live here.</p>
        <button type="button">Your first lesson is coming soon</button>
      </section>
      <p className="api-note">Connected to <code>{apiUrl}</code></p>
    </main>
  );
}
