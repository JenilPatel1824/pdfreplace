# SEO Playbook — How to Rank #1 for "PDF Text Replace"

Honest setup: the keywords you mentioned ("pdf text replace", "free pdf
text replace", "pdf edit", "edit pdf") are highly competitive. Adobe,
Smallpdf, iLovePDF, ILovePDF, PDFescape, Sejda — they all already rank.
Outranking them is a 6-12 month project, not a 1-week one. But the
**long-tail variants** are very winnable in 1-3 months.

This playbook focuses on what *actually moves rankings* in 2026.

## What's already done in this codebase

- ✅ Server-rendered HTML (Google indexes this much better than SPAs).
- ✅ `<title>`, meta description, canonical, OG tags on every page.
- ✅ JSON-LD structured data: `WebApplication`, `FAQPage`, `Article`.
- ✅ `robots.txt` + `sitemap.xml` auto-generated.
- ✅ Page speed: tiny CSS, no JS framework, async AdSense, preconnects.
- ✅ Mobile-first responsive layout.
- ✅ Semantic HTML (`<article>`, `<header>`, `<main>`, `<details>`).

## Phase 1 — Index your site (week 1)

1. Buy your domain, deploy, set `SITE_URL`.
2. Verify ownership in [Google Search Console](https://search.google.com/search-console)
   (DNS TXT record method).
3. Submit `https://yourdomain.com/sitemap.xml` in Search Console.
4. Same in [Bing Webmaster Tools](https://www.bing.com/webmasters/).
5. Confirm `/robots.txt` and `/sitemap.xml` are reachable publicly.

In Search Console, watch the "Pages" report. Within 1-2 weeks every page
should be marked "Indexed."

## Phase 2 — Long-tail content (weeks 2-12)

This is the highest-leverage activity. You can't outrank Adobe on "edit
pdf", but you absolutely can rank #1 on:

- "how to replace text in a pdf without acrobat"
- "free pdf text replace no signup"
- "how to change date in pdf invoice"
- "replace text in scanned pdf" (after OCR feature)
- "how to fix typo in a pdf for free"
- "best free pdf editor for chromebook"
- "edit pdf on iphone free"

Each one of these has 100-2000 monthly searches. Stack 30 of them and
you're at 30k+ monthly visits.

### How to write a ranking article

- **Target one specific phrase per article.** Put it in the URL, the
  `<title>`, the H1, and the first 100 words. Don't keyword-stuff —
  one mention each is enough.
- **Match search intent.** Someone searching "how to..." wants steps.
  Someone searching "best..." wants a comparison. Open Google, type
  the phrase, look at the first 5 results, and write something better.
- **1500+ words for commercial topics.** Google's ranking model
  strongly favors comprehensive content for "how-to" and "best" queries.
- **Add a screenshot per major step.** Real screenshots > stock images.
- **Internal-link to /editor.** Every article should have at least 3
  links into the editor. That's how you turn readers into users.
- **External-link to authoritative sources** (Adobe docs, MDN, etc).
  Outbound links to good sources actually help SEO.

Drop new articles in `texts/site.json` under `articles.<slug>` and
register the route in `main.go`. The template + sitemap update
automatically.

### Content cadence

Aim for 2 articles/week for 3 months = 24 articles. That's enough to
establish topical authority and start ranking on long-tails.

## Phase 3 — Backlinks (months 2-6)

Backlinks from real sites are still the #1 ranking signal Google uses.
Without backlinks your articles will plateau around position 15-25.

How to get them, ordered by ROI:

1. **Be listed in "free PDF tools" roundups.** Search
   `"best free pdf tools" 2026` and email each blog asking for
   inclusion. Conversion rate is ~5%.
2. **Reddit / Hacker News.** Post the tool to r/productivity,
   r/sysadmin, r/foss, r/Chromebook, HN. One front-page hit = 10-30
   backlinks. Don't be spammy — share in threads where your tool
   genuinely answers a question.
3. **Product Hunt launch.** Free, gets you 1-3 high-authority backlinks.
4. **Guest posts** on PDF / productivity blogs. Slow but evergreen.
5. **Help A Reporter Out (HARO)** for tech journalist quotes.

Avoid: paid link farms, comment-spam links, PBNs. Google detects them
and penalizes the whole site.

## Phase 4 — Technical SEO polish (ongoing)

Run these tools monthly and fix what they flag:

- [PageSpeed Insights](https://pagespeed.web.dev/) — aim for 95+ mobile.
- [Lighthouse](https://developer.chrome.com/docs/lighthouse) — same.
- Search Console "Core Web Vitals" report — keep all pages "Good."
- Search Console "Mobile Usability" — fix any flagged pages.

The biggest wins after launch are usually:
- Compress / lazy-load any large images.
- Add `width`/`height` to every `<img>` to prevent layout shift.
- Self-host fonts instead of Google Fonts.

## Phase 5 — Conversion (months 3+)

Ranking is half the battle. The other half is keeping users.

Track in Google Analytics 4 (free):
- **Bounce rate per page.** Pages above 70% need rewriting.
- **Time on /editor.** Users dropping off in <30s = upload UX issue.
- **/editor → /download conversion rate.** This is your North Star.
  Below 30% means something in the find/replace flow is confusing.

A/B test the hero copy, CTA button text, and ad placement.

## What NOT to bother with

- **Meta keywords tag.** Google has ignored it for 10+ years.
- **Keyword density tools.** Outdated; write naturally.
- **Submitting to 100 directories.** All low-quality, no value.
- **AI-generated bulk articles.** Google's helpful-content update
  punishes them. Write fewer, better articles.

## TL;DR

1. Ship → Search Console → sitemap submitted (week 1).
2. Write 30 long-tail articles (months 1-3).
3. Get listed in PDF-tool roundups + post to communities (months 2-4).
4. Polish Core Web Vitals + iterate based on Search Console data.
5. By month 6 you should have ~10k organic monthly visits if you
   stayed consistent. By month 12, 50k+ is realistic.
