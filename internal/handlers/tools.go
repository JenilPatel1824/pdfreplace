// Tool handlers — multipart upload + invoke pdfops + serve a download URL.
//
// Naming convention: each tool has a GET page (rendered template) and a
// POST API endpoint. The page is registered in main.go; the API is
// registered there too. Storage layout per request:
//
//   storage/tools/<id>/in_*.pdf       uploaded source(s)
//   storage/tools/<id>/output.pdf     single-output tools
//   storage/tools/<id>/parts/<n>.pdf  multi-output tools (split)
//   storage/tools/<id>/parts.zip      zipped multi-output bundle

package handlers

import (
	"archive/zip"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"pdfrep/internal/pdfops"
)

const maxToolUploadBytes = 50 << 20  // 50 MB single file
const maxMergeFiles = 20             // total files for merge
const maxMergeBytes = 100 << 20      // 100 MB combined

// --- shared helpers --------------------------------------------------------

func (a *App) toolDir(id string) string {
	return filepath.Join(a.StorageDir, "tools", id)
}

// saveUploadedPDF reads a *.pdf field from a multipart form and writes
// it under storage/tools/<id>/<name>. Returns the on-disk path.
func saveUploadedPDF(r *http.Request, field, dir, name string) (string, error) {
	f, hdr, err := r.FormFile(field)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if !strings.EqualFold(filepath.Ext(hdr.Filename), ".pdf") {
		return "", errors.New("must be a .pdf")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dst := filepath.Join(dir, name)
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, f); err != nil {
		return "", err
	}
	return dst, nil
}

// serveDownload returns a saved file under the tool dir to the caller
// with proper download headers and a friendly filename.
func (a *App) serveToolDownload(w http.ResponseWriter, r *http.Request, id, fileName, friendly string) {
	if !safeID(id) {
		http.NotFound(w, r)
		return
	}
	p := filepath.Join(a.toolDir(id), fileName)
	if _, err := os.Stat(p); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentTypeFor(fileName))
	w.Header().Set("Content-Disposition", `attachment; filename="`+friendly+`"`)
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	http.ServeFile(w, r, p)
}

func contentTypeFor(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf":
		return "application/pdf"
	case ".zip":
		return "application/zip"
	}
	return "application/octet-stream"
}

// --- Tool: Merge -----------------------------------------------------------

// MergePDFsHandler accepts multiple "pdf" form fields (the input
// element should be `<input name="pdf" multiple>`) in the order the
// user wants them concatenated.
func (a *App) MergePDFsHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxMergeBytes)
	if err := r.ParseMultipartForm(maxMergeBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "files too large or invalid form")
		return
	}
	files := r.MultipartForm.File["pdf"]
	if len(files) < 2 {
		writeErr(w, http.StatusBadRequest, "upload at least 2 PDFs")
		return
	}
	if len(files) > maxMergeFiles {
		writeErr(w, http.StatusBadRequest, "too many files (max "+strconv.Itoa(maxMergeFiles)+")")
		return
	}

	id := uuid.NewString()
	dir := a.toolDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "storage error")
		return
	}

	var paths []string
	for i, h := range files {
		if !strings.EqualFold(filepath.Ext(h.Filename), ".pdf") {
			writeErr(w, http.StatusBadRequest, h.Filename+" is not a PDF")
			return
		}
		f, err := h.Open()
		if err != nil {
			writeErr(w, http.StatusBadRequest, "could not open "+h.Filename)
			return
		}
		dst := filepath.Join(dir, "in_"+strconv.Itoa(i)+".pdf")
		out, err := os.Create(dst)
		if err != nil {
			f.Close()
			writeErr(w, http.StatusInternalServerError, "storage error")
			return
		}
		if _, err := io.Copy(out, f); err != nil {
			f.Close()
			out.Close()
			writeErr(w, http.StatusInternalServerError, "write error")
			return
		}
		f.Close()
		out.Close()
		paths = append(paths, dst)
	}

	outFile := filepath.Join(dir, "output.pdf")
	if err := pdfops.MergePDFs(paths, outFile); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/merged.pdf",
	})
}

// --- Tool: Split -----------------------------------------------------------

// SplitPDFHandler accepts:
//   - pdf:    multipart file
//   - mode:   "per_page" | "every_n" | "at_pages"
//   - n:      integer (for every_n)
//   - at:     comma-separated page numbers (for at_pages)
// Returns a download URL pointing at a zip of all parts, plus the part list.
func (a *App) SplitPDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	mode := pdfops.SplitMode(r.FormValue("mode"))
	n, _ := strconv.Atoi(r.FormValue("n"))
	atRaw := strings.TrimSpace(r.FormValue("at"))
	var at []int
	if atRaw != "" {
		for _, s := range strings.Split(atRaw, ",") {
			s = strings.TrimSpace(s)
			if s == "" {
				continue
			}
			v, err := strconv.Atoi(s)
			if err != nil || v < 2 {
				writeErr(w, http.StatusBadRequest, "split page numbers must be integers ≥ 2")
				return
			}
			at = append(at, v)
		}
	}

	partsDir := filepath.Join(dir, "parts")
	parts, err := pdfops.SplitPDF(in, partsDir, mode, n, at)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	zipPath := filepath.Join(dir, "parts.zip")
	if err := zipFiles(parts, zipPath); err != nil {
		writeErr(w, http.StatusInternalServerError, "zip: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"parts":    len(parts),
		"download": "/download-tool/" + id + "/split.zip",
	})
}

// --- Tool: Remove pages ----------------------------------------------------

// RemovePagesHandler — pdf + pages ("1,3-5,9").
func (a *App) RemovePagesHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	pages := strings.TrimSpace(r.FormValue("pages"))
	if pages == "" {
		writeErr(w, http.StatusBadRequest, "specify pages to remove (e.g. 1,3-5)")
		return
	}
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.RemovePages(in, out, pages); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/trimmed.pdf",
	})
}

// --- Tool: Remove empty pages ----------------------------------------------

func (a *App) RemoveEmptyPagesHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out := filepath.Join(dir, "output.pdf")
	removed, err := pdfops.RemoveEmptyPages(in, out)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"removed":  removed,
		"download": "/download-tool/" + id + "/cleaned.pdf",
	})
}

// --- Tool: Rotate ---------------------------------------------------------

func (a *App) RotatePDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	angle, _ := strconv.Atoi(r.FormValue("angle"))
	if angle == 0 {
		angle = 90
	}
	pages := strings.TrimSpace(r.FormValue("pages"))
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.RotatePDF(in, out, angle, pages); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/rotated.pdf",
	})
}

// --- Tool: Compress -------------------------------------------------------

func (a *App) CompressPDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.CompressPDF(in, out); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	srcInfo, _ := os.Stat(in)
	dstInfo, _ := os.Stat(out)
	srcSize, dstSize := int64(0), int64(0)
	if srcInfo != nil {
		srcSize = srcInfo.Size()
	}
	if dstInfo != nil {
		dstSize = dstInfo.Size()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          id,
		"sourceBytes": srcSize,
		"outputBytes": dstSize,
		"download":    "/download-tool/" + id + "/compressed.pdf",
	})
}

// --- Download endpoint -----------------------------------------------------

// DownloadTool serves /download-tool/{id}/{friendly}. Friendly name is
// only used for the download filename; the actual file picked depends on
// what's in the tool dir (output.pdf or parts.zip).
func (a *App) DownloadTool(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	friendly := r.PathValue("filename")
	if friendly == "" {
		friendly = "output.pdf"
	}
	// Pick the actual stored file based on extension of friendly.
	stored := "output.pdf"
	if strings.HasSuffix(strings.ToLower(friendly), ".zip") {
		stored = "parts.zip"
	}
	a.serveToolDownload(w, r, id, stored, friendly)
}

// --- Tool: Protect --------------------------------------------------------

func (a *App) ProtectPDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	password := r.FormValue("password")
	if password == "" {
		writeErr(w, http.StatusBadRequest, "password is required")
		return
	}
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.ProtectPDF(in, out, password); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/protected.pdf",
	})
}

// --- Tool: Unlock ---------------------------------------------------------

func (a *App) UnlockPDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	password := r.FormValue("password")
	if password == "" {
		writeErr(w, http.StatusBadRequest, "password is required")
		return
	}
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.UnlockPDF(in, out, password); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/unlocked.pdf",
	})
}

// --- Tool: Redact ---------------------------------------------------------

func (a *App) RedactPDFHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxToolUploadBytes)
	if err := r.ParseMultipartForm(maxToolUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}
	id := uuid.NewString()
	dir := a.toolDir(id)
	in, err := saveUploadedPDF(r, "pdf", dir, "input.pdf")
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	textToRedact := r.FormValue("textToRedact")
	if textToRedact == "" {
		writeErr(w, http.StatusBadRequest, "text to redact is required")
		return
	}
	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.RedactPDF(in, out, textToRedact); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/redacted.pdf",
	})
}

// --- helpers ---------------------------------------------------------------

func zipFiles(files []string, zipPath string) error {
	out, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	for _, f := range files {
		if err := addFileToZip(zw, f); err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(zw *zip.Writer, path string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	w, err := zw.Create(filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

// --- Tool: Organize -------------------------------------------------------

func (a *App) OrganizePDFHandler(w http.ResponseWriter, r *http.Request) {
	uploadID := strings.TrimSpace(r.FormValue("uploadId"))
	if !safeID(uploadID) {
		writeErr(w, http.StatusBadRequest, "invalid upload id")
		return
	}

	in := filepath.Join(a.StorageDir, "uploads", uploadID, "input.pdf")
	if _, err := os.Stat(in); err != nil {
		writeErr(w, http.StatusBadRequest, "uploaded file not found")
		return
	}

	pagesStr := strings.TrimSpace(r.FormValue("pages"))
	if pagesStr == "" {
		writeErr(w, http.StatusBadRequest, "pages are required")
		return
	}
	
	var selectedPages []string
	for _, p := range strings.Split(pagesStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			selectedPages = append(selectedPages, p)
		}
	}
	
	if len(selectedPages) == 0 {
		writeErr(w, http.StatusBadRequest, "no pages selected")
		return
	}

	// Create a tool dir for the output
	id := uuid.NewString()
	dir := a.toolDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeErr(w, http.StatusInternalServerError, "storage error")
		return
	}

	out := filepath.Join(dir, "output.pdf")
	if err := pdfops.OrganizePDF(in, out, selectedPages); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       id,
		"download": "/download-tool/" + id + "/organized.pdf",
	})
}
