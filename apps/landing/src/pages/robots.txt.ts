import { config } from "../data/config";

export function GET() {
  const sitemapUrl = new URL(
    "sitemap-index.xml",
    `${config.siteUrl.replace(/\/$/, "")}/`,
  ).href;

  return new Response(`User-agent: *\nAllow: /\n\nSitemap: ${sitemapUrl}\n`, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
    },
  });
}
