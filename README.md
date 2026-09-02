# Nutka

Nutka is a music-teaching starter monorepo with an Astro landing page, a React learner app, and a Go/PocketBase backend.

## Prerequisites

- Node.js 22.19+ and npm 10+
- Go 1.25+

## Start development

```sh
npm install
npm run dev
```

Outside Conductor, the services start at:

- Landing page: `http://127.0.0.1:4321`
- Learner app: `http://127.0.0.1:5173`
- PocketBase and API: `http://127.0.0.1:8090`

Inside Conductor, `CONDUCTOR_PORT` is used for the landing page and the app and backend use the next two ports, so parallel workspaces do not conflict.

The PocketBase dashboard is available at `/_/` on the backend URL after its initial setup. Runtime data is intentionally local and ignored by Git.
PocketBase also provides the API health check at `/api/health` and enables permissive development CORS by default.

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
