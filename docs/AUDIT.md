# SEO + AdSense Compliance Audit (post-tools rollout)

Snapshot of what's compliant and what still needs work, after the
PDF-tools expansion. Use this as the pre-deploy and pre-AdSense-apply
checklist.

## SEO — Google Search Central best practices

Sources:
- https://developers.google.com/search/docs/fundamentals/seo-starter-guide
- https://developers.google.com/search/docs/fundamentals/get-started
- https://developers.google.com/search/docs/appearance/structured-data

### ✅ Compliant

| Item | Where |
|---|---|
| Unique `<title>` per page | `texts/site.json` `articles[*].seo.title`, `texts/tools.json` `tools[*].seo.title`, `texts/legal.json`, `texts/seo.json` |
| Unique `<meta description>` | same JSON files, `seo.description` |
| Single `<h1>` per page that matches user intent | every template |
| Logical `<h2>` / `<h3>` hierarchy | `tool.html` "How it works", "Use cases", "FAQ"; `article.html`, `legal.html` |
| Descriptive URLs | `/merge-pdf`, `/remove-empty-pages`, etc. — no query strings |
| HTTPS enforced in prod | Cloudflare + Fly certs |
| Mobile-first responsive layout | media queries in `app.css`, dropzone + grid both adapt |
| Page speed: gzip middleware | `withGzip` in `main.go` |
| Page speed: long-cache static | `withStaticCache(31536000)` in `main.go` |
| Page speed: preconnects | `<link rel="preconnect">` for AdSense + GTM in `layout.html` |
| `robots.txt` + sitemap pointer | `internal/seo/seo.go` `Robots()` |
| Comprehensive `sitemap.xml` | `internal/seo/seo.go` `Sitemap()` — now includes /tools + 6 tool URLs |
| Canonical URL on every page | `layout.html` `<link rel="canonical">` from `CanonicalURL` in `app.go` |
| Open Graph + Twitter Card | `layout.html` `head` partial |
| `Organization` JSON-LD | `layout.html` head |
| `WebSite` JSON-LD | `layout.html` head |
| `WebApplication` JSON-LD | `layout.html` `jsonld_software` partial |
| Per-tool `SoftwareApplication` JSON-LD | `tool.html` |
| Per-tool `HowTo` JSON-LD | `tool.html` |
| Per-tool `FAQPage` JSON-LD | `tool.html` |
| `BreadcrumbList` JSON-LD | `tool.html`, `tools.html` |
| `CollectionPage` JSON-LD | `tools.html` |
| `Article` JSON-LD on guides | `article.html` |
| `alt` text on tool icons / images | inline SVGs use `aria-hidden` where decorative; `<img>` get descriptive alt |
| Internal links between related pages | tool sidebar lists 6 cross-links; site footer lists all major routes |
| `lang="en"` on `<html>` | layout.html |
| `hreflang` (en + x-default) | layout.html |

### 📋 Recommended next

| Item | Effort | Payoff |
|---|---|---|
| **20+ long-form articles** (1500+ words each, targeted long-tail keywords) | High (ongoing) | Highest — this is the actual ranking lever |
| **OG images per tool** (1200×630 PNGs) | Low — extend `cmd/gen-og` | Medium — better CTR from social shares |
| **Image sitemap** | Low | Low — only useful if we ship visual assets |
| **Schema.org `Review` / `AggregateRating`** | Medium — needs a real review collection mechanism, no fake ratings | Medium — eligible for star-rating SERP features |
| **Web vitals beacon** to Search Console | Low | Low — aggregated CrUX data from real users is what counts |

### ❌ Avoid

These are anti-patterns Google explicitly punishes:
- ❌ Keyword-stuffed body or meta tags
- ❌ Hidden text (white-on-white, off-screen, `display:none` "for SEO")
- ❌ Doorway pages (one URL per minor variant of the same content)
- ❌ AI-generated thin/duplicate content
- ❌ Fake Review/AggregateRating schema

## AdSense — pre-application checklist

Sources:
- https://support.google.com/adsense/answer/9724
- https://support.google.com/adsense/answer/12171
- https://support.google.com/adsense/answer/12923

### ✅ Already compliant

| Requirement | Status |
|---|---|
| Custom domain on HTTPS | ✅ (do at deploy) |
| Privacy policy page | ✅ `/privacy-policy` from `texts/legal.json` |
| Terms of service page | ✅ `/terms` |
| Contact page | ✅ `/contact` |
| About page | ✅ `/about` |
| `ads.txt` mounted | ✅ `public/ads.txt` (placeholder pub ID — replace before applying) |
| `<script async>` AdSense mount | ✅ `layout.html` (placeholder client ID present — replace before applying) |
| Original useful content beyond ad slots | ✅ 4 articles + 3 use-cases + 6 tool pages + 4 legal = 17 substantive pages |
| Tools that actually work end-to-end | ✅ smoke-tested all 6 tools |
| No 404s, no placeholder text | ✅ — every route registered has matching content |
| Cookie consent mounting point | ✅ — `layout.html` has `dns-prefetch` for GTM and reserves a slot for the Funding Choices snippet |

### ⚠️ Required at deploy time

| Action | Where |
|---|---|
| Replace `ca-pub-8484309816003036` with your real publisher ID | `web/templates/layout.html` (script src + ad_slot data-ad-client) |
| Replace placeholder pub ID in ads.txt | `public/ads.txt` |
| Replace `data-ad-slot="0000000000"` with real slot IDs | once AdSense is approved and you create ad units |
| Add Funding Choices cookie banner snippet | `web/templates/layout.html` after the AdSense script |
| Set `SITE_URL`, `SITE_HOST`, `SITE_SCHEME` env vars | `fly secrets set ...` |
| Verify domain in Google Search Console | DNS TXT record |
| Submit `sitemap.xml` in Search Console | one click |

### 📋 What AdSense reviewers actually look for

(Based on consolidated public reviewer-rejection feedback.)

1. **Tools that work.** They click through and try one — if it errors or
   produces a blank file, rejection. **Mitigation:** smoke-tested all 6 tools.
2. **Real, helpful written content around the tool.** Not just "Click
   here to merge." **Mitigation:** every tool has 4-step How-it-works,
   3-4 use cases, 3+ FAQ items, all in `texts/tools.json`.
3. **Internal navigation that makes sense.** Header + footer + cross-links.
   **Mitigation:** ✅ done — header has /tools link; sidebar of every tool
   page links to all other tools; footer lists all major routes.
4. **No "lorem ipsum", no half-finished pages.** **Mitigation:** every
   route serves real content backed by JSON.
5. **Clear ownership.** About page should look genuine.
   **Action item:** review `texts/legal.json` -> `about` and make sure
   it has a real publisher name, founding year, contact email.
6. **Site has been live with some organic traffic when applying.**
   **Action item:** wait ~7 days post-deploy with Search Console showing
   non-zero impressions before clicking "Request review".

## Pre-deploy verification (run before going live)

```bash
# 1. Build cleanly
go build -o pdfreplace .

# 2. Start
./pdfreplace -addr :8080

# 3. Hit every public route — all should return 200
for path in / /editor /tools \
    /merge-pdf /split-pdf /remove-pages-pdf /remove-empty-pages /rotate-pdf /compress-pdf \
    /how-to-replace-text-in-pdf /pdf-text-replace-free /pdf-text-replace-vs-adobe-acrobat \
    /replace-text-in-invoice /find-and-replace-names-in-pdf /redact-sensitive-info-pdf \
    /privacy-policy /terms /contact /about \
    /robots.txt /sitemap.xml /ads.txt; do
  printf "%4s  %s\n" "$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080$path)" "$path"
done

# 4. Smoke-test each tool API
ID=$(curl -s -F "pdf=@sample.pdf" -F "pdf=@sample.pdf" http://localhost:8080/api/tools/merge | python3 -c "import sys,json;print(json.load(sys.stdin)['id'])")
curl -I http://localhost:8080/download-tool/$ID/merged.pdf | head -1   # 200

# 5. Validate sitemap
curl -s http://localhost:8080/sitemap.xml | xmllint --noout - && echo "sitemap OK"

# 6. Validate every JSON-LD blob renders
for path in / /editor /tools /merge-pdf; do
  echo "=== $path ==="
  curl -s http://localhost:8080$path | grep -oE 'application/ld\+json'
done
```

## Risk register

| Risk | Mitigation |
|---|---|
| AdSense rejection on first try | Pre-fix: every prereq above. Be ready to add 5+ more articles and re-apply. |
| Tools time out on big PDFs | Currently 50 MB/file cap; pdfcpu handles 100+ page docs in <5s typical. If we see real timeouts, add a goroutine + progress polling endpoint. |
| Disk fill from leaked uploads | Cleanup cron in `docs/PLAN.md`. Add as a Fly scheduled job. |
| Empty-page detector false positives (e.g. very light watermark) | Defensive: requires BOTH no-text AND ≥99% white. Manually-tested OK. |
| Pure-Go limitation: decorative font replacement | Documented in `docs/REPLACEMENT.md`; can't fix without going to MuPDF/UniPDF. |
