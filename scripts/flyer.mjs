/**
 * Generuje PDF ulotki A5 do druku z trasy /ulotka.
 *
 * Sedno całości siedzi w opcji `preferCSSPageSize`. Chrome uruchomiony z flagą
 * `--print-to-pdf` ignoruje regułę `@page { size: 152mm 214mm }` i wypluwa arkusz
 * w rozmiarze Letter. Rozmiar z CSS-a respektuje dopiero `Page.printToPDF` po CDP,
 * i tylko wtedy, gdy ta flaga jest ustawiona — stąd sterowanie przeglądarką,
 * a nie zwykłe wywołanie w wierszu poleceń.
 *
 * Użycie:
 *   npm run flyer                  pełny przebieg: build + render
 *   npm run flyer -- --skip-build  render z istniejącego apps/landing/dist
 *   npm run flyer -- --png         dodatkowo podgląd PNG w 300 DPI
 */

import { spawn } from "node:child_process";
import { createServer } from "node:http";
import { createReadStream } from "node:fs";
import { mkdir, access, stat } from "node:fs/promises";
import { constants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import puppeteer from "puppeteer-core";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const distDir = path.join(repoRoot, "apps", "landing", "dist");
const outDir = path.join(repoRoot, "apps", "landing", "dist-print");
const outPdf = path.join(outDir, "ulotka-a5-152x214.pdf");
const outPng = path.join(outDir, "ulotka-a5-152x214.png");

const host = "127.0.0.1";
const skipBuild = process.argv.includes("--skip-build");
const wantPng = process.argv.includes("--png");

// Format brutto z instrukcji Drukomatu: netto 148 × 210 mm powiększone o 2 mm spadu z każdej strony.
const SHEET_MM = { width: 152, height: 214 };
const MM_PER_CSS_PX = 25.4 / 96;
const PRINT_DPI = 300;

const chromeCandidates = [
  process.env.PUPPETEER_EXECUTABLE_PATH,
  process.env.CHROME_PATH,
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
  "/Applications/Chromium.app/Contents/MacOS/Chromium",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
  "/usr/bin/google-chrome",
  "/usr/bin/chromium",
  "/usr/bin/chromium-browser",
].filter(Boolean);

async function findChrome() {
  for (const candidate of chromeCandidates) {
    try {
      await access(candidate, constants.X_OK);
      return candidate;
    } catch {
      // następny kandydat
    }
  }
  throw new Error("Nie znaleziono przeglądarki opartej na Chromium. Wskaż ją zmienną CHROME_PATH.");
}

function run(command, args, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, { stdio: "inherit", ...options });
    child.on("error", reject);
    child.on("exit", (code) =>
      code === 0 ? resolve() : reject(new Error(`${command} zakończone kodem ${code}`)),
    );
  });
}

const contentTypes = {
  ".html": "text/html; charset=utf-8",
  ".css": "text/css; charset=utf-8",
  ".js": "text/javascript; charset=utf-8",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".webp": "image/webp",
  ".woff2": "font/woff2",
  ".woff": "font/woff",
};

/**
 * Statyczny serwer na katalogu dist. Świadomie nie używamy `astro preview`:
 * ta komenda demonizuje serwer, więc zabicie procesu potomnego zostawiłoby go
 * działającego w tle, a kolejne uruchomienie podpięłoby się pod nieaktualny build.
 */
async function serveDist() {
  const server = createServer(async (req, res) => {
    const url = new URL(req.url, `http://${host}`);
    const candidates = [
      path.join(distDir, url.pathname),
      path.join(distDir, url.pathname, "index.html"),
    ];

    for (const candidate of candidates) {
      // Zabezpieczenie przed wyjściem poza dist przez ../ w ścieżce.
      if (!candidate.startsWith(distDir)) break;
      try {
        if (!(await stat(candidate)).isFile()) continue;
      } catch {
        continue;
      }
      res.writeHead(200, {
        "Content-Type": contentTypes[path.extname(candidate)] ?? "application/octet-stream",
      });
      createReadStream(candidate).pipe(res);
      return;
    }

    res.writeHead(404).end("not found");
  });

  await new Promise((resolve, reject) => {
    server.on("error", reject);
    server.listen(0, host, resolve);
  });

  return { server, port: server.address().port };
}

let served;
let browser;

try {
  if (!skipBuild) {
    console.log("→ Buduję landing…");
    await run("npm", ["run", "build", "--workspace=@nutka/landing"], { cwd: repoRoot });
  }

  // Statyczne pliki linkują assety absolutnymi ścieżkami (/_astro/…), więc file:// odpada —
  // arkusz stylów i kroje nie doczytałyby się i PDF wyszedłby bez oprawy graficznej.
  served = await serveDist();
  const url = `http://${host}:${served.port}/ulotka`;

  const executablePath = await findChrome();
  console.log(`→ Renderuję ${url}`);
  browser = await puppeteer.launch({ executablePath, headless: true });
  const page = await browser.newPage();
  await page.goto(url, { waitUntil: "networkidle0" });

  // Kroje ładują się asynchronicznie. Bez tego Chrome potrafi wydrukować arkusz
  // złożony krojem zastępczym, co przy druku widać natychmiast.
  await page.evaluate(() => document.fonts.ready);

  await mkdir(outDir, { recursive: true });
  await page.pdf({
    path: outPdf,
    printBackground: true,
    preferCSSPageSize: true,
    width: `${SHEET_MM.width}mm`,
    height: `${SHEET_MM.height}mm`,
    margin: { top: 0, right: 0, bottom: 0, left: 0 },
  });
  console.log(`✓ ${path.relative(repoRoot, outPdf)}`);

  if (wantPng) {
    // Podgląd do obejrzenia na ekranie. Skala dobrana tak, żeby wyszło dokładnie 300 DPI.
    await page.setViewport({
      width: Math.round(SHEET_MM.width / MM_PER_CSS_PX),
      height: Math.round(SHEET_MM.height / MM_PER_CSS_PX),
      deviceScaleFactor: (PRINT_DPI * MM_PER_CSS_PX) / 25.4,
    });
    // Podgląd w przeglądarce dokłada szare tło i margines wokół kartki — do PNG-a nie mogą trafić.
    await page.evaluate(() => {
      document.body.style.padding = "0";
      document.body.style.background = "none";
    });
    await (await page.$(".sheet")).screenshot({ path: outPng });
    console.log(`✓ ${path.relative(repoRoot, outPng)} (300 DPI)`);
  }

  console.log("  Format brutto 152 × 214 mm, spad 2 mm, margines wewnętrzny 3 mm.");
} catch (error) {
  console.error(`✗ ${error.message}`);
  process.exitCode = 1;
} finally {
  await browser?.close();
  served?.server.close();
}
