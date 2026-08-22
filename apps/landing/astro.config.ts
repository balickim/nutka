import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";
import { config } from "./src/data/config";

export default defineConfig({
  site: config.siteUrl,
  integrations: [sitemap()],
  server: {
    host: "127.0.0.1",
  },
  vite: {
    plugins: [tailwindcss()],
  },
});
