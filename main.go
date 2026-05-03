package main

import (
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pdfrep/internal/handlers"
	"pdfrep/internal/seo"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "listen address")
	storage := flag.String("storage", envOr("STORAGE_DIR", "storage"), "storage dir")
	flag.Parse()

	mustDir(filepath.Join(*storage, "uploads"))
	mustDir(filepath.Join(*storage, "output"))

	tpl := template.Must(template.ParseGlob("web/templates/*.html"))

	theme := handlers.LoadJSON("colors/theme.json")
	app := &handlers.App{
		Tpl:        tpl,
		StorageDir: *storage,
		SiteText:   handlers.LoadJSON("texts/site.json"),
		EditorText: handlers.LoadJSON("texts/editor.json"),
		FAQText:    handlers.LoadJSON("texts/faq.json"),
		SEO:        handlers.LoadJSON("texts/seo.json"),
		Theme:      theme,
		ThemeCSS:   handlers.BuildThemeCSS(theme),
	}

	mux := http.NewServeMux()

	// Pages
	mux.HandleFunc("GET /{$}", app.Index)
	mux.HandleFunc("GET /editor", app.Editor)
	mux.HandleFunc("GET /how-to-replace-text-in-pdf", app.Article)
	mux.HandleFunc("GET /pdf-text-replace-free", app.Article)

	// API
	mux.HandleFunc("POST /api/upload", app.Upload)
	mux.HandleFunc("GET /api/page-image/{id}/{page}", app.PageImage)
	mux.HandleFunc("POST /api/find", app.Find)
	mux.HandleFunc("POST /api/replace", app.Replace)
	mux.HandleFunc("GET /download/{id}", app.Download)

	// SEO
	mux.HandleFunc("GET /robots.txt", seo.Robots)
	mux.HandleFunc("GET /sitemap.xml", seo.Sitemap)
	mux.HandleFunc("GET /ads.txt", serveStatic("public/ads.txt"))

	// Static
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, withSecurityHeaders(mux)); err != nil {
		log.Fatal(err)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustDir(p string) {
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", p, err)
	}
}

func serveStatic(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, path) }
}

func withSecurityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		// Allow AdSense
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"img-src 'self' data: https:; "+
				"script-src 'self' 'unsafe-inline' https://pagead2.googlesyndication.com https://www.googletagmanager.com; "+
				"frame-src https://googleads.g.doubleclick.net https://tpc.googlesyndication.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self' https://pagead2.googlesyndication.com")
		h.ServeHTTP(w, r)
	})
}
