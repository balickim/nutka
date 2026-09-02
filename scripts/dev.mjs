import concurrently from "concurrently";

const fallbackPorts = { landing: 4321, app: 5173, backend: 8090 };
const conductorBasePort = Number.parseInt(process.env.CONDUCTOR_PORT ?? "", 10);
const usesConductorPorts = Number.isInteger(conductorBasePort) && conductorBasePort > 0;
const ports = usesConductorPorts
  ? {
      landing: conductorBasePort,
      app: conductorBasePort + 1,
      backend: conductorBasePort + 2,
    }
  : fallbackPorts;

const host = "127.0.0.1";
const backendUrl = `http://${host}:${ports.backend}`;

console.log(
  `Starting Nutka: landing ${ports.landing}, app ${ports.app}, backend ${ports.backend}`,
);

const { result } = concurrently(
  [
    {
      command: `npm run dev --workspace=@nutka/landing -- --port ${ports.landing}`,
      name: "landing",
      prefixColor: "cyan",
      env: { ...process.env, PUBLIC_APP_URL: `http://${host}:${ports.app}` },
    },
    {
      command: `npm run dev --workspace=@nutka/app -- --port ${ports.app}`,
      name: "app",
      prefixColor: "magenta",
      env: {
        ...process.env,
        VITE_API_URL: "/",
        VITE_DEV_API_TARGET: backendUrl,
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
