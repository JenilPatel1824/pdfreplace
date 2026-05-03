# Google AdSense — Setup & Optimization

This site is wired for Google AdSense. To go from "code is ready" to
"earning revenue" takes ~2-6 weeks because Google reviews every site
manually and won't approve until you have content + traffic.

## Phase 0 — Prerequisites (before applying)

AdSense rejects most sites on the first try. To pass on the first try:

- ✅ **Custom domain** (`yourdomain.com`, not a free subdomain).
- ✅ **HTTPS** with a valid certificate.
- ✅ **At least 10–15 pages of useful content.** Our editor + 2 article
  pages + FAQ is borderline. Add 5–10 more articles before applying.
- ✅ **Privacy policy** page. AdSense requires this — they will reject
  without it. Use a generator like [termsfeed.com](https://www.termsfeed.com/)
  and link it in the footer.
- ✅ **Cookie consent banner** for EU/UK visitors. Use [Funding Choices](https://fundingchoices.google.com/start/)
  (Google's free tool) — it's the AdSense-compatible CMP.
- ✅ **Some real traffic.** AdSense looks for organic visitors, not just
  you reloading. Aim for 100+ daily visitors before applying. Build
  traffic via SEO (see SEO.md) or by sharing in PDF/productivity
  communities on Reddit, Hacker News, Indie Hackers.
- ✅ **Site is "complete."** No "lorem ipsum", no broken links, no
  half-built pages. AdSense reviewers actually click around.

## Phase 1 — Apply

1. Sign up at [adsense.google.com](https://adsense.google.com/).
2. Add your domain. Google will give you a publisher ID like
   `pub-1234567890123456`.
3. Replace the placeholder in:
   - `web/templates/layout.html` (the `<script>` tag and every
     `data-ad-client` attribute) — search/replace
     `ca-pub-XXXXXXXXXXXXXXXX` with `ca-pub-<your-id>`.
   - `public/ads.txt` — change `pub-0000000000000000` to your real ID.
4. Deploy. Google's bot fetches `/ads.txt` and the AdSense script tag.
5. Wait. Reviews take 1–14 days, sometimes longer.

## Phase 2 — Create ad units

Once approved, create ad units in the AdSense dashboard. Pick the
**responsive display ad** type for each slot.

The site has these ad slots already (search the templates for
`ad_slot`):

- **Home** — between hero and features (top), between FAQ and footer
  (bottom).
- **Editor** — sidebar (top + bottom).
- **Article** — between lead and first heading, after the article body.

For each slot, Google gives you a `data-ad-slot` value. Replace
`0000000000` in `web/templates/layout.html` with that value. Use a
different value per slot so you can see which performs best in the
dashboard.

## Phase 3 — Optimization (this is where the money is)

### Layout rules

- **Above the fold matters most.** The hero ad and editor sidebar ad
  earn 60-70% of total revenue.
- **Don't show more than 3 ads above the fold.** Google penalizes
  "ad-heavy" layouts in both AdSense quality and SEO.
- **Leave whitespace around ads.** No clickable elements within ~150px
  — accidental clicks get your account banned.
- **Reserve the height** of each ad slot in CSS so the page doesn't
  jump when it loads. Our `.ad-slot { min-height: 100px }` handles
  this; raise it for sidebar ads (`min-height: 600px`).

### Editor page nuance

The editor is the highest-engagement page — users sit on it for
minutes. Two sidebar ads convert well there. But:

- **Never put an ad next to the download button.** Misclicks = ban.
- **Don't put an ad inside the page-preview area.** Ditto.
- **Show the editor before showing the result of replacement.** A user
  who hasn't completed an action yet won't tolerate ads as well as one
  who's celebrating success — so the strongest ad placement is the
  *post-replace download screen*, where users are most relaxed.

### What boosts CPM

- **Long-form articles** (1500+ words) targeting commercial keywords
  ("best free pdf editor", "how to redact pdf for free") get advertisers
  bidding higher than utility tool pages.
- **English-speaking traffic** from US/UK/CA/AU pays 5–10× the global
  average. SEO target those countries first.
- **Display + In-Article ads together** in long articles outperforms
  display alone.

### What gets you banned

- Clicking your own ads (even by accident — use AdBlock when testing).
- Asking users to click ads ("support us by clicking the ad below").
- Placing ads on pages with adult / pirated / hateful content.
- Showing ads on pages with no real content (login screens, error pages).

## Realistic earnings expectation

A free PDF tool with English organic traffic typically earns
$1.50–$5.00 per 1000 page views (RPM). To clear Google's $100 payout
threshold per month you need ~25,000–60,000 page views/month — which
SEO can deliver if you write 30+ targeted articles. See SEO.md.
