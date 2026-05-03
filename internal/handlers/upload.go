package handlers

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"pdfrep/internal/pdfops"
)

const maxUploadBytes = 25 << 20 // 25 MB

func (a *App) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	f, hdr, err := r.FormFile("pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "missing pdf field")
		return
	}
	defer f.Close()

	if filepath.Ext(hdr.Filename) != ".pdf" {
		writeErr(w, http.StatusBadRequest, "must be .pdf")
		return
	}

	id := uuid.NewString()
	dir := filepath.Join(a.StorageDir, "uploads", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "storage error")
		return
	}
	dst := filepath.Join(dir, "input.pdf")
	out, err := os.Create(dst)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "storage error")
		return
	}
	defer out.Close()
	if _, err := io.Copy(out, f); err != nil {
		writeErr(w, http.StatusInternalServerError, "write error")
		return
	}

	pages, err := pdfops.PageCount(dst)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "could not read pdf: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    id,
		"pages": pages,
	})
}

func (a *App) PageImage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	page := r.PathValue("page")
	if !safeID(id) {
		http.NotFound(w, r)
		return
	}
	pdf := filepath.Join(a.StorageDir, "uploads", id, "input.pdf")
	if _, err := os.Stat(pdf); err != nil {
		http.NotFound(w, r)
		return
	}
	img, err := pdfops.RenderPage(pdf, page)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(img)
}

func safeID(id string) bool {
	if len(id) < 8 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c == '-' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
