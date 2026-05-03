# FEATURES.md — Full PDF Tools Roadmap

This is the canonical roadmap for what the site offers, in priority order.
Each row links to a concrete tool route. Status legend:

- ✅ shipped
- 🚧 in this PR / now being built
- 📋 planned
- ⚠️ has known limits — see notes
- 🔬 research-only (not committing yet)

## Tier 0 — Core (already shipped)

| Feature | Route | Status | Notes |
|---|---|---|---|
| Find & replace text in PDF | `/editor` | ✅ | Vector + Scanned modes, font/size/color extraction, sampled-bg cover, bold/italic override, char-diff warning |
| Page preview rendering | (internal) | ✅ | pdftoppm at 110 DPI |
| OCR-aware bbox detection | (internal) | ✅ | pdftotext -bbox-layout |

## Tier 1 — PDF tools (this expansion)

The 6 most-used PDF tools across Smallpdf / iLovePDF / PDF24 / Sejda /
PDF2Go / Adobe Acrobat Online. All implemented as pure Go via pdfcpu
where possible.

| Tool | Route | Status | Backend |
|---|---|---|---|
| **Merge PDFs** — combine N files into one | `/merge-pdf` | 🚧 | pdfcpu `MergeCreateFile` |
| **Split PDF** — split by page ranges or every-N | `/split-pdf` | 🚧 | pdfcpu `SplitFile` / `SplitByPageNrFile` |
| **Remove pages** — delete specific pages | `/remove-pages-pdf` | 🚧 | pdfcpu `TrimFile` (keep complement) |
| **Remove empty/blank pages** — auto-detect | `/remove-empty-pages` | 🚧 | render→pixel-scan→trim |
| **Rotate PDF** — 90 / 180 / 270, all/some pages | `/rotate-pdf` | 🚧 | pdfcpu `RotateFile` |
| **Compress PDF** — reduce file size | `/compress-pdf` | 🚧 | pdfcpu `OptimizeFile` |

## Tier 2 — Conversion + extraction

Often-requested. pdfcpu supports them but each needs a UI.

| Tool | Route | Status |
|---|---|---|
| Extract images from PDF | `/extract-images` | 📋 |
| Extract text from PDF | `/extract-text` | 📋 |
| PDF metadata viewer | `/pdf-metadata` | 📋 |
| Images → PDF | `/images-to-pdf` | 📋 |
| PDF → images (PNG/JPEG) | `/pdf-to-images` | 📋 |
| HTML → PDF | `/html-to-pdf` | 🔬 (need wkhtmltopdf or chromium-headless) |
| Word/DOCX → PDF | `/word-to-pdf` | 🔬 (LibreOffice headless) |
| Excel → PDF | `/excel-to-pdf` | 🔬 (LibreOffice headless) |
| JSON → PDF | `/json-to-pdf` | 📋 (pdfcpu `CreateFile` accepts a DSL JSON) |

## Tier 3 — Security & forms

| Tool | Route | Status |
|---|---|---|
| Password-protect PDF | `/protect-pdf` | 📋 — pdfcpu `EncryptFile` |
| Remove PDF password | `/unlock-pdf` | 📋 — pdfcpu `DecryptFile` |
| Add watermark to PDF | `/watermark-pdf` | 📋 — pdfcpu `AddWatermarksFile` |
| Crop PDF | `/crop-pdf` | 📋 — pdfcpu `CropFile` |
| N-up (booklet/poster) | `/n-up-pdf` | 📋 — pdfcpu `NUpFile` |
| Sign PDF (drawn signature) | `/sign-pdf` | 🔬 |
| Fill PDF form | `/fill-pdf-form` | 📋 — pdfcpu `FillFormFile` |
| Add page numbers | `/add-page-numbers` | 📋 — text watermark per page |

## Tier 4 — Improvements to text replace

The big ones discussed in [docs/REPLACEMENT.md](docs/REPLACEMENT.md):

| Improvement | Status | What it gets us |
|---|---|---|
| Embedded-font reuse | 🔬 | Use the source PDF's actual font for the replacement (script, decorative, custom). Needs us to extract the font binary from the source, install via `api.InstallFonts`, then reference by BaseFont name in the watermark. |
| OCR for scanned PDFs without text layer | 🔬 | Tesseract integration so the tool also works on bare scans. Adds ~30s per page processing time. |
| Multi-language find (RTL, CJK) | 📋 | Proper Unicode handling for Arabic/Hebrew/Chinese/Japanese. |
| Whole-page redact | 📋 | Black bar instead of replacing |

## Tier 5 — Live editor (out of scope for v2)

A WYSIWYG visual editor where users click on text and drag it around
is its own product. Tools like PDFescape / Sejda / Smallpdf Pro have
this — it requires:
- A canvas-rendered page view (PDF.js)
- Hit-testing against text spans
- Editing UI (font picker, size, color)
- Re-emitting the page content stream with edits

This is 4–8 weeks of work. We'll revisit after Tier 1+2 ship and we
have ad revenue funding it.

---

## SEO + AdSense compliance

### Google Search Central best practices — checklist

(Source: https://developers.google.com/search/docs/fundamentals/seo-starter-guide
and https://developers.google.com/search/docs/fundamentals/get-started)

- [x] Unique `<title>` per page, ~50–60 chars
- [x] Unique `<meta name="description">` per page, ~150–160 chars
- [x] Single `<h1>` per page that matches user search intent
- [x] Logical `<h2>`/`<h3>` hierarchy
- [x] `alt` text on all `<img>`
- [x] Descriptive URL slugs (`/merge-pdf` not `/p?id=42`)
- [x] Internal links between related pages (every tool links to /editor and to other tools)
- [x] HTTPS in production
- [x] Mobile-first responsive layout
- [x] Page-speed: gzip middleware ✓ ; long-cache static ✓ ; preconnects ✓
- [x] `robots.txt` with sitemap pointer
- [x] `sitemap.xml` (auto-updated when routes change)
- [x] Canonical URL `<link rel="canonical">` on every page
- [x] Open Graph + Twitter Card tags
- [x] JSON-LD structured data: `WebApplication`, `FAQPage`, `Article`
- [ ] **Per-tool `SoftwareApplication` JSON-LD** with offers + featureList — adding now
- [ ] **`BreadcrumbList` JSON-LD** on tool pages — adding now
- [ ] **HowTo JSON-LD** on each tool's how-it-works section — adding now
- [ ] Image sitemap (low priority)
- [ ] Hreflang (only if multi-language)

### Google AdSense — pre-application requirements

(Source: https://support.google.com/adsense/answer/9724)

- [x] Custom domain with HTTPS
- [x] Privacy policy page
- [x] Terms of service page
- [x] Contact page
- [x] About page
- [x] `ads.txt` placeholder ready (replace publisher ID before applying)
- [x] Cookie consent banner mounting point (Funding Choices snippet to be added)
- [x] Original, useful content (4 articles + 3 use-cases + tool pages = ~15 pages of unique content after this PR)
- [ ] **More long-form articles** (target: 20+ for solid approval rate) — ongoing
- [ ] Site has been live for ~7+ days with organic traffic when applying — calendar
- [ ] No broken pages, no placeholder text — verified pre-deploy
- [ ] Real publisher ID swapped into `public/ads.txt` and `web/templates/layout.html` — at submission time

### What the site looks like to AdSense reviewers

Reviewers click around looking for "is this a real, polished, useful site or
is it AI slop / a thin scraper / under construction?" Things they reward:

1. **Tools that actually work** — they often try one. Make sure every
   advertised tool runs end-to-end.
2. **Helpful written content around each tool** — at least 3-4 paragraphs
   on what the tool does, when to use it, alternatives, examples. Not just
   "Click here to merge PDFs."
3. **Internal navigation** that makes sense — header + footer + cross-links.
4. **No 404s or empty pages**. Don't ship routes you haven't filled.
5. **Clear ownership** — the About page should look real, not generated.

---

## How research informed this list

The Tier-1 tool selection was determined by checking the homepage
top-bar of:

- iLovePDF (https://www.ilovepdf.com/) — Merge, Split, Compress, Convert, Rotate, Unlock, Protect, Watermark
- Smallpdf (https://smallpdf.com/) — Same set, slightly different ordering
- PDF24 (https://tools.pdf24.org/) — More extensive: 30+ tools
- Sejda (https://www.sejda.com/) — Includes blank-page detection ✓
- PDF2Go (https://www.pdf2go.com/) — Same plus image conversion

The features in **Tier 1** are the consistent intersection of all five
sites' "above the fold" tool list — the ones every PDF user expects.

Tier-4 text replacement improvements come from public discussion of
PDF text replacement on Stack Overflow / pdfcpu issues / unidoc.io blog,
plus the official PDF 1.7 spec (text content streams, font handling).
