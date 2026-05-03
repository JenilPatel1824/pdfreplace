package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"pdfrep/internal/pdfops"
)

type findReq struct {
	ID  string `json:"id"`
	Old string `json:"old"`
}

// Find scans the PDF and returns, per page, the bounding boxes
// of every match for `old`. The frontend uses these to draw
// highlight overlays so the user can confirm before replacing.
func (a *App) Find(w http.ResponseWriter, r *http.Request) {
	var req findReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if !safeID(req.ID) || strings.TrimSpace(req.Old) == "" {
		writeErr(w, http.StatusBadRequest, "id and old are required")
		return
	}
	pdf := filepath.Join(a.StorageDir, "uploads", req.ID, "input.pdf")
	hits, err := pdfops.FindText(pdf, req.Old)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"matches": hits})
}
