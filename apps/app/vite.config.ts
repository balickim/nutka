import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { tanstackRouter } from "@tanstack/router-plugin/vite";

const devApiTarget = process.env.VITE_DEV_API_TARGET ?? "http://127.0.0.1:8090";

export default defineConfig({
  // The first milestone keeps its route tree explicit in src/router.tsx;
  // generation is enabled when file-based learner routes are introduced.
  plugins: [tanstackRouter({ enableRouteGeneration: false }), react()],
  server: {
    host: "127.0.0.1",
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
