package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sort"
)


type App struct {
	Tpl        *template.Template
	StorageDir string
	SiteText   map[string]any
	EditorText map[string]any
	FAQText    map[string]any
	SEO        map[string]any
	Theme      map[string]any
	ThemeCSS   template.CSS // pre-rendered :root { --x: ... } block
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
	data["CanonicalURL"] = canonical(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.Tpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func canonical(r *http.Request) string {
	scheme := "https"
	if v := os.Getenv("SITE_SCHEME"); v != "" {
		scheme = v
	} else if r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "" {
		scheme = "http"
	}
	host := os.Getenv("SITE_HOST")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host + r.URL.Path
}

func (a *App) Index(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "index.html", map[string]any{
		"FAQ": a.FAQText["items"],
		"SEO": a.SEO["home"],
	})
}

func (a *App) Editor(w http.ResponseWriter, r *http.Request) {
	a.render(w, r, "editor.html", map[string]any{
		"Editor": a.EditorText,
		"SEO":    a.SEO["editor"],
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
