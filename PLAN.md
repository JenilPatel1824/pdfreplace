# PLAN.md — From Code to Live Site with Ads

Full ordered checklist to take this codebase from your laptop to a
revenue-earning live website. Calendar time: 2–4 weeks. Actual hands-on
work: about 8–12 hours spread across that window. AdSense approval is
what eats the calendar — Google takes 1–14 days to review, and you need
some real organic traffic before applying.

Tick boxes as you go. Sections marked **🚦 Decision** ask you to pick
one option; everything else is straight execution.

---

## Phase 0 — Prep (Day 0, 30 minutes)

Before touching anything external, make sure the code runs cleanly on
your machine and you have basic accounts ready.

- [ ] `go run .` works locally. Open http://localhost:8080, do an
      end-to-end upload → find → replace → download.
- [ ] Have a **Google account** ready (you'll use it for AdSense and
      Search Console).
- [ ] Have a **credit/debit card** ready (~$10 for the domain — you can
      run hosting on free tiers).
- [ ] Pick a **brand name + slug**. Keep it short, memorable, and
      ideally tied to "pdf"/"replace"/"edit". Examples: `pdftextreplace`,
      `pdfreplace`, `editpdftext`, `pdfreplace.app`.

---

## Phase 1 — Buy a domain (Day 0, 15 minutes, ~$10/year)

A custom domain is a **hard requirement** for AdSense and a major
ranking signal for Google. Don't try to apply with a `*.fly.dev` URL.

### 🚦 Decision: Pick a registrar

| Registrar | Why pick it | Year-1 cost |
|---|---|---|
| **Cloudflare Registrar** | At-cost pricing, no upsell, free DNS, easy SSL | ~$9 .com |
| **Porkbun** | Cheap renewals, no upsell games, simple UI | ~$10 .com |
| **Namecheap** | Long-time popular, good support | ~$13 .com |

Skip GoDaddy unless you already have an active deal there — renewals
get expensive and the upsell flow is aggressive.

### Steps

- [ ] Open the registrar's site, search for your chosen name.
- [ ] If `.com` is taken, try `.app` (TLS is enforced, looks modern) or
      `.io` (developer/tech vibe). Avoid `.xyz`, `.online`, `.site`.
- [ ] **Buy 1 year**. Don't pay for "WHOIS privacy" extras — Cloudflare/
      Porkbun include privacy free. Don't buy email hosting upsells.
- [ ] Note your domain. From now on **`yourdomain.com`** in this guide
      means the actual name you bought.

### Set up Cloudflare DNS (highly recommended)

Even if you bought elsewhere, route DNS through Cloudflare for free
SSL, CDN, and DDoS protection.

- [ ] Sign up at https://dash.cloudflare.com → **Add a site** →
      enter your domain.
- [ ] Cloudflare gives you 2 nameservers (e.g. `aria.ns.cloudflare.com`).
      In your registrar's dashboard, replace the existing nameservers
      with Cloudflare's two. Save.
- [ ] DNS propagation takes 5 min – 24 h. You'll get an email when
      Cloudflare confirms.
- [ ] In Cloudflare → **SSL/TLS** → set to **Full**, leave it on Full
      until your origin has its own cert, then upgrade to **Full
      (strict)**.
- [ ] **SSL/TLS → Edge Certificates → Always Use HTTPS: ON**.
- [ ] **Speed → Optimization → Auto Minify** (HTML/CSS/JS): all on.
- [ ] **Speed → Optimization → Brotli**: on.

> If you used Cloudflare Registrar, DNS is already on Cloudflare —
> skip the nameserver step.

---

## Phase 2 — Pick a hosting option (Day 0, 1–2 hours)

### 🚦 Decision: One of these three

| Option | Why | Cost | Best for |
|---|---|---|---|
| **Fly.io** | Free tier covers small traffic, scales to zero, simple deploys | $0–$5/mo at low traffic | Solo project / first launch |
| **VPS** (Hetzner/DigitalOcean/Vultr) | Full control, predictable cost | €4-6/mo | If you want to learn ops |
| **Cloud Run** (Google) | Scales to zero, generous free tier | $0 typically | If you're already on GCP |

Picking **Fly.io** below — it's the lowest-friction. The other two are
covered in [docs/DEPLOY.md](docs/DEPLOY.md).

### Fly.io setup

- [ ] Install flyctl:
      ```bash
      curl -L https://fly.io/install.sh | sh
      ```
- [ ] `fly auth signup` — credit card required for verification but
      the free tier covers low-traffic apps.
- [ ] Create a `Dockerfile` in the project root:

      ```dockerfile
      FROM golang:1.22-alpine AS build
      WORKDIR /src
      COPY go.mod go.sum ./
      RUN go mod download
      COPY . .
      RUN CGO_ENABLED=0 go build -o /out/pdfrep .

      FROM alpine:3.20
      RUN apk add --no-cache poppler-utils ca-certificates
      WORKDIR /app
      COPY --from=build /out/pdfrep /app/pdfrep
      COPY web        /app/web
      COPY texts      /app/texts
      COPY colors     /app/colors
      COPY public     /app/public
      EXPOSE 8080
      ENV ADDR=:8080
      CMD ["/app/pdfrep"]
      ```

- [ ] Create a minimal `fly.toml` next to it:

      ```toml
      app = "pdfrep-yourname"   # must be globally unique on Fly
      primary_region = "iad"     # or "fra", "sin", whichever is closest

      [http_service]
      internal_port = 8080
      force_https = true
      auto_stop_machines = true
      auto_start_machines = true
      min_machines_running = 0

      [[vm]]
      size = "shared-cpu-1x"
      memory = "512mb"
      ```

- [ ] First deploy:
      ```bash
      fly launch --no-deploy --copy-config   # accept defaults
      fly secrets set \
        SITE_URL=https://yourdomain.com \
        SITE_HOST=yourdomain.com \
        SITE_SCHEME=https
      fly deploy
      ```

- [ ] Verify it works at the Fly URL (e.g.
      `https://pdfrep-yourname.fly.dev`).

### Point your domain at Fly

- [ ] Tell Fly to issue certs for your domain:
      ```bash
      fly certs create yourdomain.com
      fly certs create www.yourdomain.com
      fly certs show yourdomain.com   # shows the DNS records to add
      ```
- [ ] In **Cloudflare DNS**, while Fly issues certs:
      - **temporarily** set the proxy status of those records to
        **DNS only** (grey cloud, not orange).
      - Add the **A** and **AAAA** records Fly told you to add.
- [ ] Wait 1–5 min, then run `fly certs show yourdomain.com` until
      "Certificate Status: Issued".
- [ ] Switch the Cloudflare DNS record proxy back to **Proxied**
      (orange cloud) for free CDN + DDoS.
- [ ] Verify https://yourdomain.com loads. CSS should render, the
      hero gradient should be visible, /editor should show the upload
      drop zone.

### Cleanup cron (so old uploads don't fill the disk)

- [ ] Add a tiny cleanup script that runs hourly. Easiest: use Fly's
      cron-like scheduled jobs, OR keep the binary stateless and rely
      on Fly's ephemeral filesystem (machine restarts wipe it).
      For a simple cron on a VPS:
      ```bash
      0 * * * * find /opt/pdfrep/storage -mindepth 1 -maxdepth 2 -type d -mmin +60 -exec rm -rf {} +
      ```

---

## Phase 3 — AdSense prerequisites (Day 1, 1–2 hours)

AdSense rejects ~70% of first applications. The site needs to look
**finished and trustworthy** before you apply.

### Required pages

- [ ] **Privacy policy.** Generate one at
      https://www.termsfeed.com/privacy-policy-generator (free for
      basic). Pick "Web app", "Display ads", "Google AdSense", "Cookies",
      and your country. Copy the result. Add it to the site:
      1. In `texts/site.json`, under `articles`, add a new entry:
         ```json
         "privacy": {
           "h1": "Privacy Policy",
           "lead": "How we handle your data when you use PDF Text Replace.",
           "sections": [
             {"h": "What we collect", "p": "<paste para here>"},
             {"h": "Cookies and ads", "p": "<paste para here>"}
             // …rest of the generator output
           ],
           "seo": {
             "title": "Privacy Policy — PDF Text Replace",
             "description": "Our privacy policy.",
             "keywords": "privacy policy"
           }
         }
         ```
      2. In `main.go`, add the route:
         ```go
         mux.HandleFunc("GET /privacy", app.Article)
         ```
      3. In `texts/site.json` → `footer.links`, add
         `{"label": "Privacy", "href": "/privacy"}`.
      4. In `internal/seo/seo.go` → `Sitemap`, add
         `{"/privacy", "0.5", "yearly"}`.
      5. `fly deploy`.

- [ ] **Terms of service.** Same flow at
      https://www.termsfeed.com/terms-conditions-generator. Add as
      `/terms`. Same 5 sub-steps as Privacy.

- [ ] **Contact.** A simple `/contact` page or a `mailto:` in the
      footer. Easiest: add `<li><a href="mailto:hello@yourdomain.com">Contact</a></li>`
      to `texts/site.json` → `footer.links`.

- [ ] **Cookie consent banner** for EU/UK visitors. AdSense requires
      this to serve personalised ads. Use Google's free
      [Funding Choices](https://fundingchoices.google.com/start/):
      1. Sign up with the same Google account you'll use for AdSense.
      2. Click **Start a privacy & messaging campaign** → **GDPR / TCF**.
      3. They give you a `<script>` snippet. Paste it into
         `web/templates/layout.html` immediately after the AdSense
         script tag.
      4. Deploy.

### Content depth

AdSense reviewers want to see a real site, not 4 placeholder pages.

- [ ] **Add at least 8–10 articles** before applying. Each should be
      ~1500 words, useful, written for humans not keywords. See the
      Phase 4 list for ideas.
- [ ] Each article goes in `texts/site.json` → `articles.<slug>` and
      gets a route in `main.go` and a sitemap entry.
- [ ] `fly deploy` after each batch.

---

## Phase 4 — Drive enough traffic to apply (Days 2–14)

AdSense looks at organic visitors. Aim for **~100/day for ~7 days**
before clicking apply. Most of that comes from search + community
sharing.

### Search Console setup

- [ ] Verify your site at https://search.google.com/search-console.
      Pick "Domain" property type, add the TXT record they give you in
      Cloudflare DNS, click Verify.
- [ ] Submit `https://yourdomain.com/sitemap.xml` in Search Console →
      Sitemaps.
- [ ] Same in https://www.bing.com/webmasters/.

### Write 8–10 long-tail articles

Each article targets ONE specific search phrase that has demand but
not too much competition. Suggested phrases to start with:

- [ ] "how to replace text in pdf without acrobat"
- [ ] "how to fix typo in pdf for free"
- [ ] "edit pdf on iphone free"
- [ ] "edit pdf on chromebook free"
- [ ] "best free alternative to adobe acrobat for editing pdf"
- [ ] "how to change date in pdf invoice"
- [ ] "how to update name on pdf certificate"
- [ ] "free pdf editor that works in browser no signup"
- [ ] "how to replace text in scanned pdf" (if/when you add OCR)
- [ ] "pdf find and replace online tool review"

Each article should:
- Use the target phrase in the URL, `<title>`, H1, and first paragraph.
- Be 1500+ words.
- Include 2–3 screenshots from your editor.
- Internal-link to `/editor` at least 3 times.

### Get 3–5 backlinks

- [ ] Email 5 PDF/productivity blogs that publish "best free PDF tools"
      roundups. Search Google: `"best free pdf tools" 2026`. A polite
      one-paragraph email asking for inclusion gets ~5–10% reply rate.
- [ ] Post to **r/productivity**, **r/sysadmin**, **r/foss**,
      **r/Chromebook** — only in threads where someone is genuinely
      asking how to edit a PDF. Don't spam.
- [ ] Submit to **Product Hunt** (free).
- [ ] Post to **Indie Hackers** "Show IH" thread.
- [ ] Post to **Hacker News** "Show HN" — only if you genuinely have
      a story to tell about how you built it.

---

## Phase 5 — Apply to AdSense (Day 14ish)

Don't apply earlier than this. Premature applications get rejected
and you have to wait again.

- [ ] Sign up at https://adsense.google.com.
- [ ] Add your domain. Google gives you a publisher ID like
      `pub-1234567890123456`.
- [ ] **Update three places** in the codebase:

  1. `public/ads.txt` — replace `pub-0000000000000000`:
     ```
     google.com, pub-1234567890123456, DIRECT, f08c47fec0942fa0
     ```

  2. `web/templates/layout.html` — find/replace **every occurrence** of
     `ca-pub-XXXXXXXXXXXXXXXX` with `ca-pub-1234567890123456`. There
     are several: the `<script>` tag, every `<ins data-ad-client=>`.

  3. (You'll do this in Phase 6 once approved.) The
     `data-ad-slot="0000000000"` placeholders.

- [ ] `fly deploy`.
- [ ] Verify https://yourdomain.com/ads.txt shows the real publisher ID.
- [ ] Verify the `<script>` tag with your real `ca-pub-...` is in the
      page source of https://yourdomain.com/.
- [ ] In AdSense → click **Request review**.
- [ ] Wait 1–14 days. Don't keep clicking re-review — that resets the
      queue.

### If rejected

- [ ] Read the email carefully. Common reasons:
      - **"Insufficient content"** → write 5 more articles, wait 2
        weeks, re-apply.
      - **"Doesn't comply with policies"** → check Privacy + Terms are
        present and linked. Check there's no broken or empty page.
      - **"Site under construction"** → likely something's broken.
        Click around your live site, find anything that 404s or
        renders empty, fix it.
- [ ] Fix and re-apply. There's no penalty for re-applying.

---

## Phase 6 — Once approved (1–7 days after applying)

- [ ] In AdSense → **Ads → By ad unit → Create new ad unit**. Create:
      1. **"Home top"** — type: Display, format: Responsive.
      2. **"Article inline"** — type: Display, format: Responsive.
      3. **"Editor sidebar"** — type: Display, format: Responsive.
- [ ] Each one gives you a `data-ad-slot` value (a 10-digit number).
- [ ] In `web/templates/layout.html` → the `{{define "ad_slot"}}`
      block currently has `data-ad-slot="0000000000"` — replace with
      one of your real slot IDs.
- [ ] **Optional but better**: differentiate the slots. Right now all
      `{{template "ad_slot" .}}` use the same block. Split into 3
      blocks (`ad_top`, `ad_inline`, `ad_sidebar`) each with its own
      `data-ad-slot`, then update the templates to use the right one
      per location. You'll get per-slot performance data in AdSense.
- [ ] `fly deploy`.
- [ ] Wait **24–48 hours** for ads to actually appear. Google needs
      time to crawl your pages and decide what to serve.

### Verify ads are loading

- [ ] In a browser **without** ad blocker (use incognito + disable
      extensions), open the homepage. There should be visible ads in
      the slot positions.
- [ ] Open DevTools → Network → filter "googlesyndication". You should
      see the ad scripts loading.
- [ ] Don't ever click your own ads. Even one accidental click can
      ban your account permanently.

---

## Phase 7 — Optimize (ongoing)

In rough order of payoff:

1. **Write more articles.** Each new long-tail article adds to your
   organic traffic floor. Aim for 2/week for the first 3 months.
2. **Watch Search Console weekly.** Look at "Performance" → "Queries"
   to see what searches are hitting your site. Write new articles
   targeting the queries you're already showing up for at position
   10–25 (those are the ones that move quickly with effort).
3. **Watch AdSense weekly.** Compare RPM (revenue per 1000 views)
   across your 3 ad slots. The lowest-performing one — try moving it
   or removing it. Don't keep dead slots.
4. **Watch Core Web Vitals** in Search Console. Anything tagged
   "Needs improvement" or "Poor" hurts both rankings and ad revenue.
   Most common culprit: large images. Use `width`/`height` attrs on
   every `<img>`.
5. **A/B test the hero CTA.** Change the button copy or color in
   `texts/site.json` / `colors/theme.json`, deploy, measure
   /editor traffic over 2 weeks.

### Red lines that get you banned

- Clicking your own ads (even once, even by accident)
- Asking users to click ads ("support us!")
- Putting ads on pages with adult/pirated/hateful content
- Putting ads close enough to the download button that users misclick
  → users complain → AdSense reviews → ban

---

## Money — what to actually expect

Free PDF tool with English organic traffic:

| Time since launch | Realistic monthly revenue |
|---|---|
| Month 1 | $0 (no ads serving yet, and tiny traffic) |
| Month 2–3 | $5–$30 |
| Month 4–6 | $30–$150 |
| Month 6–12 | $100–$500 |
| Year 2+ | $300–$1500 if you keep writing articles |

You won't get rich on a free PDF tool, but a focused, well-SEO'd one
can pay its hosting + a coffee budget after ~6 months. The cap depends
mostly on how many quality articles you write — articles compound,
hosting cost doesn't.

---

## Reference docs (deeper detail per topic)

- [docs/DOMAIN.md](docs/DOMAIN.md) — registrar comparison, DNS records.
- [docs/DEPLOY.md](docs/DEPLOY.md) — Fly.io, VPS+nginx, Cloud Run all in detail.
- [docs/ADS.md](docs/ADS.md) — placement tactics, what boosts CPM, what gets you banned.
- [docs/SEO.md](docs/SEO.md) — long-tail keyword strategy, backlinks, what NOT to do.
- [docs/REPLACEMENT.md](docs/REPLACEMENT.md) — when/how to upgrade the PDF replacement engine.
- [docs/GETTING_LIVE.md](docs/GETTING_LIVE.md) — earlier version of this doc; PLAN.md
  supersedes it.

---

## TL;DR — minimum path to first dollar

1. Buy domain on Cloudflare Registrar — 15 min, $10.
2. Deploy on Fly.io — 1 hour, $0.
3. Add Privacy + Terms + Contact — 30 min, $0.
4. Write 8 articles — 16 hours over 2 weeks, $0.
5. Submit sitemap to Search Console, post to 3 communities — 1 hour.
6. Apply to AdSense — 5 min, wait 1–14 days.
7. Add ad slot IDs once approved — 15 min.
8. Wait. Optimize. Repeat the article cycle.

Total hands-on: **~20 hours**. First revenue: **~6–12 weeks** out.
