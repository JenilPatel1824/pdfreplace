# From Code → Live Site → AdSense Revenue

The full ordered checklist. Each box is a single concrete action — tick
them in order. Whole path takes 1–4 weeks of calendar time, but only
~6–10 hours of actual work.

---

## Phase 1 — Get a domain (Day 0, 15 min, ~$10/year)

- [ ] Pick a domain. Suggested:
      `pdftextreplace.com`, `pdfreplace.app`, `editpdftext.com`,
      `pdfreplace.io`. **Stick to .com / .app / .io** — they look
      trustworthy and don't have spam baggage.
- [ ] Buy at **Cloudflare Registrar** (best value, no upsell games)
      or **Porkbun**. Avoid GoDaddy.
- [ ] In Cloudflare DNS, leave it on Cloudflare nameservers — you'll
      use Cloudflare for SSL + CDN later.

> Why first? A real custom domain is a hard requirement for AdSense
> approval and for SEO ranking. Without it, the rest is wasted.

---

## Phase 2 — Deploy the site (Day 0, 1–2 hours, free or ~$5/mo)

Pick **one** path. Easiest first.

### Option A — Fly.io (recommended for solo project)

- [ ] Install flyctl: `curl -L https://fly.io/install.sh | sh`
- [ ] `fly auth signup` (free tier covers small traffic)
- [ ] In the project root, create `Dockerfile` (copy from
      [DEPLOY.md](./DEPLOY.md)).
- [ ] `fly launch --no-deploy` → answer prompts, accept defaults.
- [ ] `fly secrets set SITE_URL=https://yourdomain.com SITE_HOST=yourdomain.com SITE_SCHEME=https`
- [ ] `fly deploy`
- [ ] `fly certs create yourdomain.com` and `fly certs create www.yourdomain.com`
- [ ] Add the DNS A/AAAA records Fly tells you to (in Cloudflare DNS,
      set **Proxy status: DNS only** while Fly issues certs, then
      switch back to **Proxied** once certs are live).

### Option B — VPS (Hetzner / DigitalOcean / Vultr, ~$4-6/mo)

Use this if you want full control. See [DEPLOY.md](./DEPLOY.md) for the
full systemd + nginx + certbot recipe.

### Option C — Cloud Run (Google) / Render / Railway

All work with the same Dockerfile. Cloud Run scales to zero — pay-per-use.

### After deploy (any option)

- [ ] Verify `https://yourdomain.com/` loads and CSS renders correctly.
- [ ] Verify `https://yourdomain.com/robots.txt` shows your sitemap URL.
- [ ] Verify `https://yourdomain.com/sitemap.xml` lists all pages.
- [ ] In Cloudflare: SSL → **Full (strict)**, **Always Use HTTPS** on,
      **Auto Minify** on (HTML/CSS/JS).

---

## Phase 3 — Required for AdSense (Day 1, 1 hour)

AdSense rejects ~70% of first-time applications. Doing these
prerequisites before applying gets you into the 30% that pass.

- [ ] **Privacy policy.** Generate one at
      [termsfeed.com](https://www.termsfeed.com/) — answer 5 questions,
      copy the result. Add it as `/privacy` (drop a new article in
      `texts/site.json` under `articles.privacy` and register the route
      in `main.go`). Link it from the footer.
- [ ] **Terms of service.** Same — generate, add as `/terms`, link in footer.
- [ ] **Contact page or email.** Add a `mailto:` in the footer or a
      `/contact` page.
- [ ] **Cookie consent banner** for EU/UK visitors. Use Google's free
      [Funding Choices](https://fundingchoices.google.com/start/) — sign
      up, paste their snippet into `web/templates/layout.html` right
      after the AdSense script.
- [ ] **More content.** AdSense reviewers look for ~15+ pages of
      genuinely useful content. You currently have 4 (home, editor, 2
      articles). Add 10+ more articles. Suggestions in the next phase.

---

## Phase 4 — Drive enough traffic to apply (Days 2–14, 10–20 hours)

AdSense looks for organic visitors. ~100/day for 1–2 weeks is the
informal threshold for approval.

- [ ] **Write 10 long-tail articles.** Each ~1500 words, each targeting
      a specific search phrase (full list of phrase ideas in
      [SEO.md](./SEO.md) Phase 2). Add them as new entries in
      `texts/site.json` → `articles` and register routes in `main.go`.
- [ ] **Verify in Google Search Console.** Add property
      `https://yourdomain.com`, verify via DNS TXT record (Cloudflare
      DNS).
- [ ] Submit `https://yourdomain.com/sitemap.xml` in Search Console.
- [ ] Same for [Bing Webmaster Tools](https://www.bing.com/webmasters/).
- [ ] **Share in 3 communities** where the tool genuinely answers a
      question:
      - Reddit: r/productivity, r/sysadmin, r/foss, r/Chromebook
      - Hacker News: only if the launch post is genuinely interesting
      - Indie Hackers: "Show IH" thread
      - Product Hunt: free launch
- [ ] **Get 1–3 backlinks.** Email 5 "free PDF tools" roundup blogs
      asking for inclusion. Search:
      `"best free pdf tools" 2026 site:medium.com`.

---

## Phase 5 — Apply to AdSense (Day 14)

- [ ] Sign up at [adsense.google.com](https://adsense.google.com).
- [ ] Add your domain. Google gives you a publisher ID like
      `pub-1234567890123456`.
- [ ] **Update 3 places** with that ID:
      1. `public/ads.txt` → replace `pub-0000000000000000`.
      2. `web/templates/layout.html` → replace **every** occurrence of
         `ca-pub-XXXXXXXXXXXXXXXX` (the script tag and every
         `data-ad-client` attribute).
      3. Re-deploy (`fly deploy` or whatever your host).
- [ ] Verify `https://yourdomain.com/ads.txt` shows your real ID.
- [ ] In AdSense: click **Request review**. Wait 1–14 days.

> If rejected, the email tells you why. The most common reasons:
>   - "Insufficient content" → write more articles.
>   - "Site doesn't comply with policies" → add privacy + terms pages.
>   - "Site under construction" → fix any broken links / missing pages.
> Fix and re-apply. There's no penalty for re-applying.

---

## Phase 6 — Once approved (1–7 days after applying)

- [ ] In AdSense, create **3 ad units**:
      - "Home top" — Display, Responsive
      - "Article inline" — Display, Responsive
      - "Editor sidebar" — Display, Responsive
- [ ] Replace `data-ad-slot="0000000000"` in
      `web/templates/layout.html` with your real slot IDs (you can
      use a different slot for each placement to compare which
      performs best).
- [ ] Re-deploy.
- [ ] Wait 24–48h for ads to actually start serving — Google needs
      time to crawl and decide what to show.

---

## Phase 7 — Optimize (ongoing)

Read these in this order as you grow:

1. [SEO.md](./SEO.md) — long-tail keyword strategy, backlinks, what NOT to do.
2. [ADS.md](./ADS.md) — placement rules, what gets you banned, what boosts CPM.
3. [REPLACEMENT.md](./REPLACEMENT.md) — when to upgrade the PDF
   replacement engine (PyMuPDF / UniPDF) for better font preservation.

---

## Money expectations — be realistic

- Months 1–3: $0–$5/month while traffic builds.
- Months 4–6: $20–$80/month if you wrote 30+ articles.
- Months 6–12: $100–$500/month is realistic for a focused free PDF tool
  with English organic traffic.
- Plateaus are normal — they break when you add new articles or when
  Google updates its ranking signals.

You won't get rich on a free PDF tool, but a well-SEO'd one can pay
its hosting bill and a coffee budget within ~6 months.

---

## Troubleshooting

- **`fly deploy` complains about poppler.** The Dockerfile in
  [DEPLOY.md](./DEPLOY.md) uses `apk add poppler-utils` — make sure
  that line is present.
- **AdSense ads not showing after approval.** Wait 48h. Then check the
  browser console: AdBlock, CSP, or ad-blocker extensions on your test
  device commonly block them. Try in incognito.
- **Cloudflare 525/526 errors.** Set SSL to **Full (strict)** but only
  after the origin has a valid cert. Initially set **Full** while certs
  are issuing.
- **Search Console shows "Discovered – currently not indexed".** Normal
  for the first 1–2 weeks; Google takes time. Keep writing content.
