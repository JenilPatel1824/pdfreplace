package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
)


type App struct {
	Tpl        *template.Template
	StorageDir string
	SiteText   map[string]any
	EditorText map[string]any
	FAQText    map[string]any
	SEO        map[string]any
	UseCases   map[string]any
	LegalPages map[string]any
	ToolsText  map[string]any // texts/tools.json — tool landing + per-tool copy
	Theme      map[string]any
	ThemeCSS   template.CSS   // pre-rendered :root { --x: ... } block
}

// BuildThemeCSS turns the tokens map from colors/theme.json into a
// CSS string, sorted for deterministic output. Returned as template.CSS
// so html/template will not escape the leading "--" of each variable.
func BuildThemeCSS(theme map[string]any) template.CSS {
	tokens, ok := theme["tokens"].(map[string]any)
	if !ok {
		return ""
	}
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b []byte
	b = append(b, ":root{"...)
	for _, k := range keys {
		v, _ := tokens[k].(string)
		b = append(b, fmt.Sprintf("%s:%s;", k, v)...)
	}
	b = append(b, '}')
	return template.CSS(b)
}

func LoadJSON(path string) map[string]any {
	b, err := os.ReadFile(path)
	if err != nil {
		log.Printf("warn: cannot read %s: %v", path, err)
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		log.Printf("warn: invalid json %s: %v", path, err)
		return map[string]any{}
	}
	return m
}

func (a *App) render(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	data["Site"] = a.SiteText
	data["Theme"] = a.Theme
	data["ThemeCSS"] = a.ThemeCSS
	if _, ok := data["SEO"]; !ok {
		data["SEO"] = a.SEO["home"]
	}
	origin := publicOrigin(r)
	data["SiteOrigin"] = origin
	data["CanonicalURL"] = origin + canonicalPath(r.URL.Path)
	data["OGImage"] = ogImagePath(data["SEO"])
	if _, ok := data["OGType"]; !ok {
		data["OGType"] = "website"
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.Tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

// publicOrigin is scheme + host (no path), for absolute URLs in OG/Twitter tags.
func publicOrigin(r *http.Request) string {
	scheme := "https"
	if v := os.Getenv("SITE_SCHEME"); v != "" {
		scheme = v
	} else if fp := r.Header.Get("X-Forwarded-Proto"); fp == "http" || fp == "https" {
		scheme = fp
	} else if r.TLS == nil {
		scheme = "http"
	}
	host := os.Getenv("SITE_HOST")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

// canonicalPath normalizes the URL path for <link rel="canonical"> (no query string).
// Host canonicalization (www vs apex) is via SITE_HOST and your CDN/DNS.
func canonicalPath(raw string) string {
	if raw == "" || raw == "/" {
		return "/"
	}
	return path.Clean("/" + strings.TrimPrefix(raw, "/"))
}

func ogImagePath(seo any) string {
	m, ok := seo.(map[string]any)
	if !ok {
		return "/static/og-home.png"
	}
	v, _ := m["ogImage"].(string)
	if v != "" {
		return v
	}
	return "/static/og-home.png"
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "index.html", map[string]any{
		"FAQ":       a.FAQText["items"],
		"SEO":       a.SEO["home"],
		"Tools":     a.toolsList(),
		"ToolsCats": a.ToolsText["index"], // for category headings
		"OGType":    "website",
	})
}

// toolsList returns the tools as a stable-ordered slice ([{slug,...}]),
// sorted by slug for deterministic rendering. Templates can range over
// it with positional indices (used for ItemList JSON-LD).
func (a *App) toolsList() []map[string]any {
	tools, _ := a.ToolsText["tools"].(map[string]any)
	slugs := make([]string, 0, len(tools))
	for k := range tools {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	out := make([]map[string]any, 0, len(slugs))
	for _, k := range slugs {
		t, ok := tools[k].(map[string]any)
		if !ok {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (a *App) Editor(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "editor.html", map[string]any{
		"Editor": a.EditorText,
		"SEO":    a.SEO["editor"],
		"OGType": "website",
	})
}

func (a *App) Article(w http.ResponseWriter, r *http.Request) {
	slug := r.URL.Path[1:]
	articles, _ := a.SiteText["articles"].(map[string]any)
	art, _ := articles[slug].(map[string]any)
	if art == nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "article.html", map[string]any{
		"Article": art,
		"SEO":     art["seo"],
		"OGType":  "article",
	})
}

func (a *App) UseCase(w http.ResponseWriter, r *http.Request, slug string) {
	uc, _ := a.UseCases[slug].(map[string]any)
	if uc == nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "article.html", map[string]any{
		"Article": uc,
		"SEO":     uc["seo"],
		"OGType":  "article",
	})
}

func (a *App) Legal(w http.ResponseWriter, r *http.Request, id string) {
	page, _ := a.LegalPages[id].(map[string]any)
	if page == nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "legal.html", map[string]any{
		"Page":   page,
		"SEO":    page["seo"],
		"OGType": "website",
	})
}

// ToolsIndex renders /tools — the catalogue of every PDF tool.
func (a *App) ToolsIndex(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "tools.html", map[string]any{
		"Index":  a.ToolsText["index"],
		"Tools":  a.ToolsText["tools"],
		"SEO":    a.ToolsText["seo_index"],
		"OGType": "website",
	})
}

// ToolPage renders one tool — slug must match a key under
// texts/tools.json -> tools[slug].
func (a *App) ToolPage(w http.ResponseWriter, r *http.Request, slug string) {
	tools, _ := a.ToolsText["tools"].(map[string]any)
	tool, _ := tools[slug].(map[string]any)
	if tool == nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "tool.html", map[string]any{
		"Tool":   tool,
		"Slug":   slug,
		"SEO":    tool["seo"],
		"OGType": "website",
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
