import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";
import tailwindcss from "@tailwindcss/vite";
import { config } from "./src/data/config";
import { DEV_HOST, DEV_PORTS } from "../../scripts/ports.mjs";

const siteUrl = new URL(config.siteUrl);
const base =
  siteUrl.pathname === "/" || process.env.NODE_ENV === "development" ? undefined : siteUrl.pathname;

export default defineConfig({
  site: siteUrl.origin,
  base,
  // /ulotka to arkusz do druku, nie strona dla odwiedzających — poza mapą witryny.
  integrations: [sitemap({ filter: (page) => !page.includes("/ulotka") })],
  server: {
    host: DEV_HOST,
    port: DEV_PORTS.landing,
  },
  vite: {
    // Fail loudly instead of drifting to another port when 5321 is taken.
    server: { strictPort: true },
    plugins: [tailwindcss()],
  },
});
