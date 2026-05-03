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
	"time"
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
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	now := time.Now().UTC().Format("2006-01-02")
	base := siteURL()

	urls := []struct {
		Path     string
		Priority string
		Freq     string
	}{
		{"/", "1.0", "weekly"},
		{"/editor", "0.9", "weekly"},
		{"/how-to-replace-text-in-pdf", "0.8", "monthly"},
		{"/pdf-text-replace-free", "0.8", "monthly"},
	}

	fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?>`)
	fmt.Fprint(w, `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, u := range urls {
		fmt.Fprintf(w,
			`<url><loc>%s%s</loc><lastmod>%s</lastmod><changefreq>%s</changefreq><priority>%s</priority></url>`,
			base, u.Path, now, u.Freq, u.Priority,
		)
	}
	fmt.Fprint(w, `</urlset>`)
}
