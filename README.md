# Nutka

Nutka is a music-teaching starter monorepo with an Astro landing page, a React scheduling app, and a Go/PocketBase backend.

The landing page and the app share one visual language from `packages/ui` (`@nutka/ui`): design tokens, self-hosted fonts, base typography, and framework-free component classes such as buttons, the brand mark, and the eyebrow label.

## Prerequisites

- Node.js 22.19+ and npm 10+
- Go 1.25+

## Start development

```sh
npm install
npm run dev
```

The services always start at the same ports, inside and outside Conductor:

- Landing page: `http://127.0.0.1:5321`
- Scheduling app: `http://127.0.0.1:6173`
- PocketBase and API: `http://127.0.0.1:9090`

Each port is the tool's default port with its leading digit bumped by one, so Nutka never collides with an unrelated Astro, Vite, or PocketBase instance. `scripts/ports.mjs` is the single source of truth; the landing and app dev servers use `strictPort`, so a busy port fails instead of silently moving. Because the ports are fixed, only one workspace can run `npm run dev` at a time. The app calls the API through relative `/api` URLs and the dev server proxies them to the backend, which keeps browser auth cookies same-origin in local development.

The PocketBase dashboard is available at `/_/` on the backend URL after its initial setup. Runtime data is intentionally local and ignored by Git.
PocketBase also provides the API health check at `/api/health` and enables permissive development CORS by default.

## Authentication and local setup

Start the services with `npm run dev`, then seed verified local accounts from a second terminal:

```sh
cd apps/backend
NUTKA_ENV=development go run . seed-teacher \
  --email teacher@example.test \
  --password 'local-password' \
  --name 'Test Teacher'

NUTKA_ENV=development go run . seed-learner \
  --email learner@example.test \
  --password 'local-password' \
  --name 'Test Learner'

NUTKA_ENV=development go run . seed-assignment \
  --teacher teacher@example.test \
  --learner learner@example.test \
  --default-duration-minutes 45
```

All seed commands are development-only. Persona commands create or update verified accounts. The assignment command accepts a teacher and learner ID or email and defaults the lesson duration to 45 minutes. Commands never log passwords. There is no public registration flow.

Teacher and learner sessions use separate `__Host-nutka_teacher_session` and `__Host-nutka_learner_session` cookies. Each cookie is `HttpOnly`, `Secure`, `SameSite=Lax`, scoped to `Path=/`, host-only, and valid for 12 hours. The server never exposes the session token to JavaScript. The app restores sessions through `/api/teachers/auth/me` or `/api/learners/auth/me` and clears only the matching state on logout. See [docs/auth.md](docs/auth.md) for the complete auth contract.

Teacher routes use `/teachers/login`, `/teachers`, and `/teachers/availability`. Learner routes use `/learners/login` and `/learners/calendar`. The root route selects a persona.

The following auth features are intentionally deferred: public registration, invitation delivery, password reset, MFA, profile editing, refresh-token rotation, and localization of auth UI.

## Landing page

The landing page is Polish-only and targets adult hobby learners. Copy lives in
`apps/landing/src/data/site.ts`; contact details, prices, photos and other deployment-specific values
live in `apps/landing/src/data/config.ts`. Values that are not yet known
are marked with a `TODO:` prefix and render as-is on the page, so an unfilled field is impossible to
miss. Never replace a `TODO:` with an invented value.

Optional environment variables for the landing page:

- `PUBLIC_APP_URL` — target of the "Zaloguj się" links. Set automatically by `npm run dev`.
- `PUBLIC_UMAMI_SRC` and `PUBLIC_UMAMI_WEBSITE_ID` — Umami analytics. The script is only emitted when
  both are set, so local and preview builds stay tracking-free. Umami is cookieless, so no consent
  banner is required.

The temporary GitHub Pages deployment URL is configured in `apps/landing/src/data/config.ts`.
The sitemap and QR code are generated from it. When a real domain is selected, update `siteUrl`
and add a matching `CNAME` file under `apps/landing/public/`.

GitHub Pages deployment is configured in `.github/workflows/deploy.yml`. In the repository's
Settings → Pages, choose **GitHub Actions** as the source.

## Useful commands

```sh
npm run dev          # Start all services
npm run dev:landing  # Start only Astro
npm run dev:app      # Start only React
npm run dev:backend  # Start only PocketBase
npm run build        # Build both frontends
npm run check        # Type-check frontends and test Go
```
