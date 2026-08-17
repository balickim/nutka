import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  // TODO: podmienić na docelową domenę. Bez tego sitemap się nie wygeneruje.
  site: "https://dominikaluczyszyn.pl",
  integrations: [sitemap()],
  server: {
    host: "127.0.0.1",
  },
  vite: {
    plugins: [tailwindcss()],
  },
});
