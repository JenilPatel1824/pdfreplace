// Package seo serves robots.txt and a dynamic sitemap.
//
// The site URL must be set via the SITE_URL env var in production
// (e.g. https://pdftextreplace.com). In dev we default to localhost so
// links don't break, but Google won't index a localhost sitemap.
package seo

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

func siteURL() string {
	if v := os.Getenv("SITE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8080"
}

func Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", siteURL())
}

func Sitemap(w http.ResponseWriter, r *http.Request) {
	// Strict XML MIME for crawlers (Google Search Central / sitemap spec).
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	base := siteURL()

	// lastmod reflects real content updates (bump when you change a URL meaningfully).
	urls := []struct {
		Path     string
		Priority string
		Freq     string
		LastMod  string
	}{
		{"/", "1.0", "weekly", "2026-05-03"},
		{"/editor", "0.9", "weekly", "2026-05-03"},
		{"/how-to-replace-text-in-pdf", "0.8", "monthly", "2026-05-03"},
		{"/pdf-text-replace-free", "0.8", "monthly", "2026-05-03"},
		{"/pdf-text-replace-vs-adobe-acrobat", "0.75", "monthly", "2026-05-03"},
		{"/replace-text-in-invoice", "0.7", "monthly", "2026-05-03"},
		{"/find-and-replace-names-in-pdf", "0.7", "monthly", "2026-05-03"},
		{"/redact-sensitive-info-pdf", "0.7", "monthly", "2026-05-03"},
		{"/privacy-policy", "0.4", "yearly", "2026-05-03"},
		{"/terms", "0.4", "yearly", "2026-05-03"},
		{"/contact", "0.35", "yearly", "2026-05-03"},
		{"/about", "0.35", "yearly", "2026-05-03"},
	}

	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		fmt.Fprintf(w,
			`<url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>`,
			base, u.Path, u.LastMod, u.Freq, u.Priority,
		)
	}
	fmt.Fprint(w, `</urlset>`)
}
