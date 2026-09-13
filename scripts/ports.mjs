// Fixed development ports, shared by every entry point (npm run dev, the
// per-app dev scripts, and Conductor). Each one is the tool's default port
// with its leading digit bumped by one, so Nutka never collides with an
// unrelated Astro, Vite, or PocketBase instance.
export const DEV_HOST = "127.0.0.1";

export const DEV_PORTS = {
  landing: 5321, // astro default 4321
  app: 6173, // vite default 5173
  backend: 9090, // pocketbase default 8090
};

export const BACKEND_URL = `http://${DEV_HOST}:${DEV_PORTS.backend}`;
