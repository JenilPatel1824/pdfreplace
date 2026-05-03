package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pdfrep/internal/pdfops"
)

type replaceReq struct {
	ID     string `json:"id"`
	Old    string `json:"old"`
	New    string `json:"new"`
	Pages  []int  `json:"pages"` // 1-based; empty = all
	Mode   string `json:"mode"`  // "vector" (default) | "scanned"
	Bold   bool   `json:"bold"`  // force bold weight
	Italic bool   `json:"italic"`// force italic / oblique
}

func (a *App) Replace(w http.ResponseWriter, r *http.Request) {
	var req replaceReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !safeID(req.ID) || strings.TrimSpace(req.Old) == "" {
		writeErr(w, http.StatusBadRequest, "id and old are required")
		return
	}
	in := filepath.Join(a.StorageDir, "uploads", req.ID, "input.pdf")
	if _, err := os.Stat(in); err != nil {
		writeErr(w, http.StatusNotFound, "upload not found")
		return
	}
	outDir := filepath.Join(a.StorageDir, "output", req.ID)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "storage error")
		return
	}
	out := filepath.Join(outDir, "output.pdf")

	opts := pdfops.Options{Mode: req.Mode, ForceBold: req.Bold, ForceItalic: req.Italic}
	if err := pdfops.ReplaceWithOpts(in, out, req.Old, req.New, req.Pages, opts); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"download": "/download/" + req.ID,
	})
}

func (a *App) Download(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !safeID(id) {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(a.StorageDir, "output", id, "output.pdf")
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="replaced.pdf"`)
	http.ServeFile(w, r, p)
}
