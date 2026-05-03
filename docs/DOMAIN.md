# Getting a Domain

A keyword-rich, easy-to-remember domain helps SEO *and* click-through-rate
from search results. Aim for something short, brandable, and ideally
contains "pdf" plus one of "edit/replace/text".

## Good candidates (check availability first)

- `pdftextreplace.com`
- `pdfreplace.com`
- `editpdftext.com`
- `pdftextedit.app`
- `pdftools.app`

The exact phrase doesn't have to match — Google ranks by content quality
much more than by domain keywords today. **Pick a short, memorable name**
over a clunky keyword stuff like `free-pdf-text-replace-online.com`.

## Where to buy

Avoid the cheapest registrars — they upsell aggressively and make
transfers painful. These are reliable and reasonably priced:

| Registrar         | Why                                                |
|-------------------|----------------------------------------------------|
| **Cloudflare**    | At-cost pricing, great DNS, free SSL — best value |
| **Porkbun**       | Cheap renewals, no upsell games                   |
| **Namecheap**     | Easy UI, good support                              |

Skip GoDaddy unless you already have a deal there — renewals get expensive.

## TLD choice

- `.com` is still the gold standard for trust and recall.
- `.app` and `.io` are fine for tech tools (both require HTTPS by default — good).
- `.org` is for non-profits; using it for a commercial site looks off.
- Avoid `.xyz`, `.online`, `.site` — they carry a "spam" signal in some
  search-quality heuristics and look cheap to users.

## After purchase — DNS setup

Whichever host you use (next doc), you'll set:

```
A     @       <server-ip>
A     www     <server-ip>     # or CNAME to @
```

Or for a CDN-fronted setup (recommended):

```
CNAME @       your-app.fly.dev   (or wherever you host)
CNAME www     your-app.fly.dev
```

Then in Cloudflare, turn on:
- **SSL: Full (strict)**
- **Always Use HTTPS: on**
- **Auto Minify: on**
- **Brotli: on**

Set `SITE_URL=https://yourdomain.com` in your server env so the sitemap
and canonical URLs use the right host.
