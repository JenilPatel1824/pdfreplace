package main

import (
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pdfrep/internal/cleanup"
	"pdfrep/internal/handlers"
	"pdfrep/internal/seo"
)

func main() {
	addr := flag.String("addr", envOr("ADDR", ":8080"), "listen address")
	storage := flag.String("storage", envOr("STORAGE_DIR", "storage"), "storage dir")
	flag.Parse()

	mustDir(filepath.Join(*storage, "uploads"))
	mustDir(filepath.Join(*storage, "output"))
	mustDir(filepath.Join(*storage, "tools"))

	// Start background cleanup job:
	// Runs every 15 minutes, deletes processed files/folders older than 1 hour.
	log.Println("Starting background cleanup job (maxAge: 1 hour)...")
	cleanup.Start(*storage, 1*time.Hour, 15*time.Minute)

	funcs := template.FuncMap{
		"add":  func(a, b int) int { return a + b },
		"json": func(v any) (template.JS, error) {
			b, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return template.JS(b), nil
		},
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
	}
	tpl := template.Must(template.New("").Funcs(funcs).ParseGlob("web/templates/*.html"))

	theme := handlers.LoadJSON("colors/theme.json")
	app := &handlers.App{
		Tpl:        tpl,
		StorageDir: *storage,
		SiteText:   handlers.LoadJSON("texts/site.json"),
		EditorText: handlers.LoadJSON("texts/editor.json"),
		FAQText:    handlers.LoadJSON("texts/faq.json"),
		SEO:        handlers.LoadJSON("texts/seo.json"),
		UseCases:   handlers.LoadJSON("texts/usecases.json"),
		LegalPages: handlers.LoadJSON("texts/legal.json"),
		ToolsText:  handlers.LoadJSON("texts/tools.json"),
		Theme:      theme,
		ThemeCSS:   handlers.BuildThemeCSS(theme),
	}

	mux := http.NewServeMux()

	// Pages
	mux.HandleFunc("GET /{$}", app.Index)
	mux.HandleFunc("GET /editor", app.Editor)
	mux.HandleFunc("GET /how-to-replace-text-in-pdf", app.Article)
	mux.HandleFunc("GET /pdf-text-replace-free", app.Article)
	mux.HandleFunc("GET /pdf-text-replace-vs-adobe-acrobat", app.Article)

	for _, slug := range []string{
		"replace-text-in-invoice",
		"find-and-replace-names-in-pdf",
		"redact-sensitive-info-pdf",
	} {
		slug := slug
		mux.HandleFunc("GET /"+slug, func(w http.ResponseWriter, r *http.Request) {
			app.UseCase(w, r, slug)
		})
	}

	mux.HandleFunc("GET /privacy-policy", func(w http.ResponseWriter, r *http.Request) {
		app.Legal(w, r, "privacy-policy")
	})
	mux.HandleFunc("GET /terms", func(w http.ResponseWriter, r *http.Request) {
		app.Legal(w, r, "terms")
	})
	mux.HandleFunc("GET /contact", func(w http.ResponseWriter, r *http.Request) {
		app.Legal(w, r, "contact")
	})
	mux.HandleFunc("GET /about", func(w http.ResponseWriter, r *http.Request) {
		app.Legal(w, r, "about")
	})

	// PDF tools
	mux.HandleFunc("GET /tools", app.ToolsIndex)
	for _, slug := range []string{
		"merge-pdf",
		"split-pdf",
		"remove-pages-pdf",
		"remove-empty-pages",
		"rotate-pdf",
		"compress-pdf",
		"protect-pdf",
		"unlock-pdf",
		"redact-pdf",
		"organize-pdf",
		"pdf-to-image",
		"image-to-pdf",
	} {
		slug := slug
		mux.HandleFunc("GET /"+slug, func(w http.ResponseWriter, r *http.Request) {
			app.ToolPage(w, r, slug)
		})
	}

	// API
	mux.HandleFunc("POST /api/upload", app.Upload)
	mux.HandleFunc("GET /api/page-image/{id}/{page}", app.PageImage)
	mux.HandleFunc("POST /api/find", app.Find)
	mux.HandleFunc("POST /api/replace", app.Replace)
	mux.HandleFunc("GET /download/{id}", app.Download)

	// Tool APIs
	mux.HandleFunc("POST /api/tools/merge", app.MergePDFsHandler)
	mux.HandleFunc("POST /api/tools/split", app.SplitPDFHandler)
	mux.HandleFunc("POST /api/tools/remove-pages", app.RemovePagesHandler)
	mux.HandleFunc("POST /api/tools/remove-empty-pages", app.RemoveEmptyPagesHandler)
	mux.HandleFunc("POST /api/tools/rotate", app.RotatePDFHandler)
	mux.HandleFunc("POST /api/tools/compress", app.CompressPDFHandler)
	mux.HandleFunc("POST /api/tools/protect", app.ProtectPDFHandler)
	mux.HandleFunc("POST /api/tools/unlock", app.UnlockPDFHandler)
	mux.HandleFunc("POST /api/tools/redact", app.RedactPDFHandler)
	mux.HandleFunc("POST /api/tools/organize", app.OrganizePDFHandler)
	mux.HandleFunc("POST /api/tools/pdf-to-image", app.PDFToImageHandler)
	mux.HandleFunc("POST /api/tools/image-to-pdf", app.ImageToPDFHandler)
	mux.HandleFunc("GET /download-tool/{id}/{filename}", app.DownloadTool)

	// SEO
	mux.HandleFunc("GET /robots.txt", seo.Robots)
	mux.HandleFunc("GET /sitemap.xml", seo.Sitemap)
	mux.HandleFunc("GET /ads.txt", serveAdsTxt("public/ads.txt"))

	// Static (long cache; bump filenames or query when assets change)
	mux.Handle("GET /static/", withStaticCache(31536000, http.StripPrefix("/static/", http.FileServer(http.Dir("web/static")))))

	log.Printf("listening on %s", *addr)
	if err := http.ListenAndServe(*addr, withSecurityHeaders(withGzip(mux))); err != nil {
		log.Fatal(err)
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	w io.Writer
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	return g.w.Write(b)
}

// withGzip compresses responses when the client accepts gzip (Core Web Vitals / transfer size).
// Pair with CDN Brotli (e.g. Cloudflare) in production for even smaller payloads.
func withGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasPrefix(p, "/static/") || strings.HasPrefix(p, "/api/") {
			h.ServeHTTP(w, r)
			return
		}
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			h.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		h.ServeHTTP(gzipResponseWriter{ResponseWriter: w, w: gz}, r)
	})
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

func serveAdsTxt(filePath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.ServeFile(w, r, filePath)
	}
}

func withStaticCache(maxAgeSec int, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d, immutable", maxAgeSec))
		h.ServeHTTP(w, r)
	})
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
				"script-src 'self' 'unsafe-inline' https://pagead2.googlesyndication.com https://www.googletagmanager.com https://cdnjs.cloudflare.com; "+
				"frame-src https://googleads.g.doubleclick.net https://tpc.googlesyndication.com; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self' https://pagead2.googlesyndication.com")
		h.ServeHTTP(w, r)
	})
}
