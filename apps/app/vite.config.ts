import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";

import { BACKEND_URL, DEV_HOST, DEV_PORTS } from "../../scripts/ports.mjs";

const devApiTarget = process.env.VITE_DEV_API_TARGET ?? BACKEND_URL;

export default defineConfig({
  // The first milestone keeps its route tree explicit in src/router.tsx;
  // generation is enabled when file-based learner routes are introduced.
  plugins: [tanstackRouter({ enableRouteGeneration: false }), react()],
  server: {
    host: DEV_HOST,
    port: DEV_PORTS.app,
    // Fail loudly instead of drifting to another port when 6173 is taken.
    strictPort: true,
    proxy: {
      "/api": {
        target: devApiTarget,
        changeOrigin: false,
        secure: false,
      },
      "/api/collections": {
        target: devApiTarget,
        changeOrigin: false,
        secure: false,
      },
    },
  },
});
