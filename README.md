# pdfrep — Free PDF Text Replace

A small, SEO-optimized, ad-supported web app that lets users replace text
inside a PDF for free. Pure Go server + vanilla HTML/JS frontend, designed
to rank for "pdf text replace" / "free pdf text replace" / "edit pdf"
long-tail searches and earn revenue via Google AdSense.

## Run locally

Three steps from a fresh copy of this folder.

### 1. Install prereqs (one-time per machine)

You need **Go 1.22+** and the **poppler** binaries (`pdftotext`,
`pdftoppm`, `pdfinfo`) — the app shells out to them.

```bash
# Linux (Debian/Ubuntu)
sudo apt install golang-1.22 poppler-utils

# macOS
brew install go poppler

# Windows: install Go from https://go.dev/dl/ and poppler from
# https://github.com/oschwartz10612/poppler-windows/releases
# (add poppler/bin to PATH). WSL works too.
```

### 2. From inside the project folder

```bash
cd pdfrep              # or whatever you renamed the folder to
go mod download        # pulls pdfcpu + uuid (one-time, ~30 MB)
```

### 3. Run

```bash
# quick (compile + run in one step):
go run .

# or build a binary first:
go build -o pdfrep .
./pdfrep
```

Then open **http://localhost:8080**.

### Flags & env vars

```bash
./pdfrep -addr :3000              # different port
./pdfrep -storage /var/pdfrep     # different storage location

SITE_URL=https://yourdomain.com \
SITE_HOST=yourdomain.com \
SITE_SCHEME=https \
./pdfrep
```

### Verify it's running

```bash
curl -I http://localhost:8080/         # 200
curl -I http://localhost:8080/editor   # 200
which pdftotext pdftoppm pdfinfo       # all should resolve
```

If `pdftotext` / `pdftoppm` / `pdfinfo` aren't on PATH, find/replace
will fail with "could not read pdf" — install poppler and ensure those
binaries are in your shell's PATH.

### Stopping the server

```bash
# foreground: Ctrl+C
# background: pkill -f './pdfrep'
```

### Cleanup of uploaded/output files

The runtime stores uploads and outputs in `storage/` (auto-created).
A simple cron-friendly cleanup that deletes files older than 60 minutes:

```bash
find storage -mindepth 1 -maxdepth 2 -type d -mmin +60 -exec rm -rf {} +
```

No DB, no Redis, no external services — the binary plus the poppler
tools is the entire runtime.

## Project layout

```
main.go                  HTTP server, routing, security headers
internal/handlers/       upload / find / replace / download / pages
internal/pdfops/         page count, render-to-PNG, find bboxes, replace
internal/seo/            robots.txt + sitemap.xml
web/templates/           index / editor / article + shared partials
web/static/css|js/       app.css, editor.js
texts/                   ALL site copy lives here (JSON)
colors/theme.json        ALL colors live here (CSS vars)
public/                  ads.txt
storage/                 uploads & outputs (gitignore)
docs/                    DOMAIN, DEPLOY, SEO, ADS, REPLACEMENT
```

`texts/` and `colors/` are the only places to edit copy and colors —
the templates and CSS read from them.

## Going live

The whole path from "code on my laptop" to "live with ads earning" is
written out as a checklist in **[PLAN.md](PLAN.md)** — buy domain,
deploy, AdSense prerequisites, traffic-building, application, ad
slots. Read that first.

For deeper detail per topic see [docs/DEPLOY.md](docs/DEPLOY.md)
(Fly.io / VPS / Cloud Run), [docs/DOMAIN.md](docs/DOMAIN.md),
[docs/ADS.md](docs/ADS.md), [docs/SEO.md](docs/SEO.md).

Set these env vars in production:

```
SITE_URL=https://yourdomain.com
SITE_HOST=yourdomain.com
SITE_SCHEME=https
ADDR=:8080
```

## Customize

- **Copy & translations:** edit `texts/*.json`. Restart server to
  reload (or wire fsnotify for hot-reload — small upgrade).
- **Theme colors:** edit `colors/theme.json`.
- **AdSense:** replace `ca-pub-XXXXXXXXXXXXXXXX` in
  `web/templates/layout.html` and `pub-0000000000000000` in
  `public/ads.txt` with your publisher ID. See [docs/ADS.md](docs/ADS.md).

## Honest limitations

- v1 maps the source font to one of pdfcpu's 14 standard fonts
  (Helvetica / Times / Courier × Bold/Italic). Decorative or script
  fonts can't be reproduced; see [docs/REPLACEMENT.md](docs/REPLACEMENT.md)
  for the PyMuPDF / MuPDF / UniPDF upgrade paths.
- Pure image-only scanned PDFs (no text layer at all) can't be
  searched. PDFs with an OCR text layer work — switch the editor to
  **Scanned mode**.
- Replacement text much wider than the original may overflow. The
  editor warns yellow at 1.3× / +5 chars and red at 2× / +10 chars.
# pdfreplace
# pdfreplace
