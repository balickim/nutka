import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";
import { config } from "./src/data/config";

export default defineConfig({
  site: config.siteUrl,
  // /ulotka to arkusz do druku, nie strona dla odwiedzających — poza mapą witryny.
  integrations: [sitemap({ filter: (page) => !page.includes("/ulotka") })],
  server: {
    host: "127.0.0.1",
  },
  vite: {
    plugins: [tailwindcss()],
  },
});
