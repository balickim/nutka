import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";
import { config } from "./src/data/config";

const siteUrl = new URL(config.siteUrl);
const base =
  siteUrl.pathname === "/" || process.env.NODE_ENV === "development" ? undefined : siteUrl.pathname;

export default defineConfig({
  site: siteUrl.origin,
  base,
  // /ulotka to arkusz do druku, nie strona dla odwiedzających — poza mapą witryny.
  integrations: [sitemap({ filter: (page) => !page.includes("/ulotka") })],
  server: {
    host: "127.0.0.1",
  },
  vite: {
    plugins: [tailwindcss()],
  },
});
