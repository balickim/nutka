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

## Useful commands

```sh
npm run dev          # Start all services
npm run dev:landing  # Start only Astro
npm run dev:app      # Start only React
npm run dev:backend  # Start only PocketBase
npm run build        # Build both frontends
npm run check        # Type-check frontends and test Go
```
