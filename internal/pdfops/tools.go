// Package pdfops — multi-tool PDF operations on top of pdfcpu.
//
// Functions here are stateless: each takes input file path(s), an output
// path, and operation-specific args. The HTTP handlers in
// internal/handlers/tools.go are thin wrappers that handle uploads,
// call these, and serve the result.
//
// All operations preserve the original document's structure where
// possible — pdfcpu reads, validates, optimizes, then writes a fresh
// PDF, so output is typically smaller and more compatible than the
// input even with a passthrough operation.

package pdfops

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// MergePDFs concatenates inFiles in order into outFile. Empty inFiles is
// rejected; a single inFile is treated as a copy.
func MergePDFs(inFiles []string, outFile string) error {
	if len(inFiles) == 0 {
		return errors.New("at least one PDF is required")
	}
	if len(inFiles) == 1 {
		return copyFile(inFiles[0], outFile)
	}
	conf := model.NewDefaultConfiguration()
	if err := api.MergeCreateFile(inFiles, outFile, false, conf); err != nil {
		return fmt.Errorf("merge: %w", err)
	}
	return nil
}

// SplitMode is how SplitPDF chops the input.
//
//	SplitPerPage  — every page becomes its own file (span = 1)
//	SplitEveryN   — N pages per chunk (span = N)
//	SplitAtPages  — split before each of the listed page numbers
//	                (e.g. [3, 7] → files 1-2, 3-6, 7-end)
type SplitMode string

const (
	SplitPerPage SplitMode = "per_page"
	SplitEveryN  SplitMode = "every_n"
	SplitAtPages SplitMode = "at_pages"
)

// SplitPDF writes one or more PDF parts under outDir.
// Returns the list of file paths in the order they were produced.
//
// pdfcpu's SplitFile / SplitByPageNrFile name parts predictably; we
// scan outDir for *.pdf afterwards rather than parsing pdfcpu's stdout.
func SplitPDF(inFile, outDir string, mode SplitMode, n int, atPages []int) ([]string, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	conf := model.NewDefaultConfiguration()

	switch mode {
	case SplitPerPage:
		if err := api.SplitFile(inFile, outDir, 1, conf); err != nil {
			return nil, fmt.Errorf("split: %w", err)
		}
	case SplitEveryN:
		if n < 1 {
			return nil, errors.New("n must be >= 1")
		}
		if err := api.SplitFile(inFile, outDir, n, conf); err != nil {
			return nil, fmt.Errorf("split: %w", err)
		}
	case SplitAtPages:
		if len(atPages) == 0 {
			return nil, errors.New("at least one split page is required")
		}
		if err := api.SplitByPageNrFile(inFile, outDir, atPages, conf); err != nil {
			return nil, fmt.Errorf("split: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown split mode %q", mode)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".pdf") {
			out = append(out, outDir+"/"+e.Name())
		}
	}
	if len(out) == 0 {
		return nil, errors.New("split produced no files")
	}
	return out, nil
}

// RemovePages keeps the complement of `removeRanges` in inFile and
// writes that to outFile. `removeRanges` accepts pdfcpu page-selection
// syntax: "1,3", "2-4", "1,3-5,9".
//
// Implementation: pdfcpu has TrimFile which keeps a selection; we
// invert the user's "remove these" into "keep these" by enumerating
// all pages then filtering.
func RemovePages(inFile, outFile string, removeRanges string) error {
	total, err := PageCount(inFile)
	if err != nil {
		return fmt.Errorf("page count: %w", err)
	}
	remove, err := parsePageRanges(removeRanges, total)
	if err != nil {
		return err
	}
	if len(remove) == 0 {
		return errors.New("no pages selected to remove")
	}
	if len(remove) >= total {
		return errors.New("would remove every page; nothing left to save")
	}

	keep := make([]string, 0, total-len(remove))
	removeSet := make(map[int]bool, len(remove))
	for _, p := range remove {
		removeSet[p] = true
	}
	for p := 1; p <= total; p++ {
		if !removeSet[p] {
			keep = append(keep, strconv.Itoa(p))
		}
	}

	conf := model.NewDefaultConfiguration()
	if err := api.TrimFile(inFile, outFile, keep, conf); err != nil {
		return fmt.Errorf("trim: %w", err)
	}
	return nil
}

// RotatePDF rotates the selected pages (or all if pageRanges is empty)
// by the given angle. Angle must be 90 / 180 / 270 / -90 etc.
// (pdfcpu accepts any multiple of 90, positive or negative.)
func RotatePDF(inFile, outFile string, angle int, pageRanges string) error {
	if angle%90 != 0 {
		return errors.New("rotation angle must be a multiple of 90")
	}
	var sel []string
	if strings.TrimSpace(pageRanges) != "" {
		// pass through to pdfcpu — it accepts "1-3,5,7-9"
		sel = []string{pageRanges}
	}
	conf := model.NewDefaultConfiguration()
	if err := api.RotateFile(inFile, outFile, angle, sel, conf); err != nil {
		return fmt.Errorf("rotate: %w", err)
	}
	return nil
}

// CompressPDF runs pdfcpu's optimize pass which deduplicates resources,
// re-encodes streams, and removes unreferenced objects. Compression
// ratio depends entirely on what's inefficient in the input — typical
// gains are 5-30% on text PDFs, 0-10% on image-heavy PDFs.
//
// For aggressive image recompression you'd need an extra pass through
// ghostscript or mutool. Out of scope for now.
func CompressPDF(inFile, outFile string) error {
	conf := model.NewDefaultConfiguration()
	if err := api.OptimizeFile(inFile, outFile, conf); err != nil {
		return fmt.Errorf("compress: %w", err)
	}
	return nil
}

// RemoveEmptyPages renders every page at low DPI, decides "this page is
// blank" if (a) it has no extractable text AND (b) ≥99% of pixels are
// near-white, then removes those pages.
//
// Returns the list of page numbers detected as empty (1-based, in the
// pre-removal numbering).
func RemoveEmptyPages(inFile, outFile string) ([]int, error) {
	total, err := PageCount(inFile)
	if err != nil {
		return nil, err
	}
	if total <= 1 {
		return nil, errors.New("can't run blank-page removal on a 1-page PDF")
	}

	empty := []int{}
	for p := 1; p <= total; p++ {
		blank, err := pageLooksBlank(inFile, p)
		if err != nil {
			// Skip pages we can't inspect — don't drop them by accident.
			continue
		}
		if blank {
			empty = append(empty, p)
		}
	}
	if len(empty) == 0 {
		// Nothing blank; just copy through.
		return nil, copyFile(inFile, outFile)
	}
	if len(empty) >= total {
		return nil, errors.New("every page detected as blank — refusing to save")
	}

	// Build the keep selector.
	keepSet := make(map[int]bool, total-len(empty))
	for p := 1; p <= total; p++ {
		keepSet[p] = true
	}
	for _, p := range empty {
		delete(keepSet, p)
	}
	keep := make([]string, 0, len(keepSet))
	for p := 1; p <= total; p++ {
		if keepSet[p] {
			keep = append(keep, strconv.Itoa(p))
		}
	}

	conf := model.NewDefaultConfiguration()
	if err := api.TrimFile(inFile, outFile, keep, conf); err != nil {
		return nil, fmt.Errorf("trim: %w", err)
	}
	return empty, nil
}

// pageLooksBlank renders the page and returns true if it has no
// extractable text and ≥99% of pixels are within 5 luminance points of
// pure white.
func pageLooksBlank(inFile string, page int) (bool, error) {
	// Cheap text check first — extracting text is faster than rendering.
	hasText, err := pageHasText(inFile, page)
	if err != nil {
		return false, err
	}
	if hasText {
		return false, nil
	}

	bb, err := RenderPage(inFile, strconv.Itoa(page))
	if err != nil {
		return false, err
	}
	img, err := png.Decode(bytes.NewReader(bb))
	if err != nil {
		return false, err
	}
	bnd := img.Bounds()
	total := (bnd.Max.X - bnd.Min.X) * (bnd.Max.Y - bnd.Min.Y)
	if total == 0 {
		return false, nil
	}
	const stride = 4 // skip pixels for speed; doesn't change the outcome
	const whiteCutoff = 250
	white := 0
	checked := 0
	for y := bnd.Min.Y; y < bnd.Max.Y; y += stride {
		for x := bnd.Min.X; x < bnd.Max.X; x += stride {
			r, g, b, _ := img.At(x, y).RGBA()
			R := uint8(r >> 8)
			G := uint8(g >> 8)
			B := uint8(b >> 8)
			if R >= whiteCutoff && G >= whiteCutoff && B >= whiteCutoff {
				white++
			}
			checked++
		}
	}
	if checked == 0 {
		return false, nil
	}
	return float64(white)/float64(checked) >= 0.99, nil
}

// pageHasText returns true if pdftotext extracts any non-whitespace
// from the page.
func pageHasText(inFile string, page int) (bool, error) {
	out, err := runPdftotextPage(inFile, page)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// parsePageRanges expands "1,3-5,8" into [1,3,4,5,8] and validates
// against total page count.
func parsePageRanges(s string, total int) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("page selection is empty")
	}
	out := []int{}
	seen := make(map[int]bool)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			ab := strings.SplitN(part, "-", 2)
			a, err1 := strconv.Atoi(strings.TrimSpace(ab[0]))
			b, err2 := strconv.Atoi(strings.TrimSpace(ab[1]))
			if err1 != nil || err2 != nil || a < 1 || b < a || b > total {
				return nil, fmt.Errorf("invalid range %q", part)
			}
			for i := a; i <= b; i++ {
				if !seen[i] {
					seen[i] = true
					out = append(out, i)
				}
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > total {
				return nil, fmt.Errorf("invalid page %q", part)
			}
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out, nil
}
