import concurrently from "concurrently";

import { BACKEND_URL, DEV_HOST, DEV_PORTS } from "./ports.mjs";

const ports = DEV_PORTS;
const host = DEV_HOST;

console.log(
  `Starting Nutka: landing ${ports.landing}, app ${ports.app}, backend ${ports.backend}`,
);

const { result } = concurrently(
  [
    {
      command: `npm run dev --workspace=@nutka/landing`,
      name: "landing",
      prefixColor: "cyan",
      env: { ...process.env, PUBLIC_APP_URL: `http://${host}:${ports.app}` },
    },
    {
      command: `npm run dev --workspace=@nutka/app`,
      name: "app",
      prefixColor: "magenta",
      env: {
        ...process.env,
        VITE_API_URL: "/",
        VITE_DEV_API_TARGET: BACKEND_URL,
        VITE_LANDING_URL: `http://${host}:${ports.landing}`,
      },
    },
    {
      command: `go run . serve --http=${host}:${ports.backend}`,
      cwd: "apps/backend",
      name: "backend",
      prefixColor: "green",
    },
  ],
  {
    killOthers: ["failure"],
    prefix: "name",
  },
);

try {
  await result;
} catch (error) {
  process.exitCode = typeof error?.code === "number" ? error.code : 1;
}
