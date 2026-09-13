// Offers the two closed-account entry points without bootstrapping or mixing persona sessions.

import { Link } from "@tanstack/react-router";

export function PersonaEntryView() {
  return (
    <main className="center-shell">
      <section className="status-card" aria-labelledby="persona-entry-heading">
        <p className="wordmark">nutka</p>
        <p className="eyebrow">Przestrzeń dla muzyki</p>
        <h1 id="persona-entry-heading">Wybierz swoją przestrzeń</h1>
        <p><Link className="primary-button" to="/learners/login">Logowanie ucznia</Link></p>
        <p><Link className="secondary-button" to="/teachers/login">Logowanie nauczyciela</Link></p>
      </section>
    </main>
  );
}
